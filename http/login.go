package http

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	fiber "github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v2"
	"github.com/golang-jwt/jwt"

	"github.com/jonoton/scout/notify"
)

func getSHA256Hash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

func generateSecret() string {
	length := 6
	random := make([]byte, length)
	rand.Read(random)
	secret := fmt.Sprintf("%x", random)[:length]
	return secret
}
func getTwoFactorMaskedOptions(rxConfig *notify.RxConfig) []string {
	maskedOptions := make([]string, 0)
	for _, cur := range rxConfig.Email {
		masked := ""
		if len(cur) > 6 {
			emailTokens := strings.Split(cur, "@")
			if len(emailTokens) == 2 {
				usrStr := emailTokens[0]
				atStr := emailTokens[1]
				if len(usrStr) > 3 {
					usrStr = usrStr[:3]
				}
				masked = fmt.Sprintf("%s...@%s", usrStr, atStr)
			} else {
				masked = fmt.Sprintf("%s...%s", cur[:3], cur[len(cur)-3:])
			}
		} else if len(cur) > 0 {
			masked = fmt.Sprintf("%s...", string(cur[0])) // first element
		}
		maskedOptions = append(maskedOptions, masked)
	}
	for _, cur := range rxConfig.GetPhones() {
		phone := cur.Number
		masked := ""
		if len(phone) > 6 {
			masked = fmt.Sprintf("%s...%s", phone[:3], phone[len(phone)-3:])
		} else if len(phone) > 0 {
			masked = fmt.Sprintf("...%s", phone[len(phone)-1:]) // last element
		}
		maskedOptions = append(maskedOptions, masked)
	}
	return maskedOptions
}

type twoFactorAttempt struct {
	time   time.Time
	secret string
}

func newTwoFactorAttempt() *twoFactorAttempt {
	t := &twoFactorAttempt{
		time:   time.Now(),
		secret: generateSecret(),
	}
	return t
}

func (h *Http) validUser(user string, pass string) (bool, string) {
	if h.httpConfig == nil {
		return false, ""
	}
	for _, cur := range h.httpConfig.Users {
		if getSHA256Hash(cur.User) == user && getSHA256Hash(cur.Password) == pass {
			return true, cur.User
		}
	}
	return false, ""
}

func (h *Http) userHasTwoFactor(user string) (bool, *notify.RxConfig, int) {
	if h.httpConfig == nil {
		return false, nil, 0
	}
	for _, cur := range h.httpConfig.Users {
		if cur.User == user {
			rxConfig := cur.TwoFactor
			numEmails := len(rxConfig.Email)
			numPhones := len(rxConfig.GetPhones())
			hasTwoFactor := numEmails > 0 || numPhones > 0
			return hasTwoFactor, &rxConfig, numEmails + numPhones
		}
	}
	return false, nil, 0
}

func (h *Http) createToken(user string, timeNow time.Time) (string, error) {
	// Create token
	token := jwt.New(jwt.SigningMethodHS256)
	// Set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["user"] = user
	claims["exp"] = timeNow.Add(time.Hour * 24 * 7).Unix()
	if h.httpConfig != nil && h.httpConfig.SignInExpireDays > 0 {
		claims["exp"] = timeNow.Add(time.Hour * 24 * time.Duration(h.httpConfig.SignInExpireDays)).Unix()
	}
	// Generate encoded token
	return token.SignedString([]byte(h.loginKey))
}

func (h *Http) sendSecret(index int, rxConfig *notify.RxConfig, attempt twoFactorAttempt) {
	title := "Scout Passcode"
	html := "Scout Passcode:  " + attempt.secret
	pos := 0
	for _, cur := range rxConfig.Email {
		if pos == index {
			h.manage.Notifier.SendEmail([]string{cur}, title, html, make([]string, 0), make([]string, 0))
			return
		}
		pos++
	}
	for _, cur := range rxConfig.GetPhones() {
		if pos == index {
			h.manage.Notifier.SendText([]notify.Phone{cur}, title, html, make([]string, 0))
			return
		}
		pos++
	}
}

func (h *Http) loginHandler(c *fiber.Ctx) error {
	user := c.FormValue("a")
	pass := c.FormValue("b")
	factorIndexStr := c.FormValue("y")
	factorIndex, err := strconv.Atoi(factorIndexStr)
	if err != nil {
		factorIndex = -1
	}
	hasFactorIndex := factorIndex >= 0
	sharedSecret := c.FormValue("z")
	hasSharedSecret := len(sharedSecret) > 0
	timeNow := time.Now()
	if len(user) > 0 && len(pass) > 0 {
		if validUserPass, vUser := h.validUser(user, pass); validUserPass {
			if hasTwoFactor, rxConfig, numFactors := h.userHasTwoFactor(vUser); hasTwoFactor {
				if hasSharedSecret {
					// Secret Provided
					if userCheck, found := h.twoFactorCheck[vUser]; found && sharedSecret == userCheck.secret {
						t, err := h.createToken(user, timeNow)
						if err != nil {
							h.loginLogger.Printf("%s,error,%s,%s,%s\r\n", getFormattedKitchenTimestamp(timeNow), vUser, c.IP(), c.IPs())
							return c.SendStatus(fiber.StatusInternalServerError)
						}
						// Success Two Factor
						h.loginLogger.Printf("%s,success,%s,%s,%s\r\n", getFormattedKitchenTimestamp(timeNow), vUser, c.IP(), c.IPs())
						return c.JSON(fiber.Map{"c": t})
					} else {
						// Fail Two Factor
						delete(h.twoFactorCheck, vUser)
						h.loginLogger.Printf("%s,bad secret,%s,%s,%s\r\n", getFormattedKitchenTimestamp(timeNow), vUser, c.IP(), c.IPs())
						return c.SendStatus(fiber.StatusUnauthorized)
					}
				} else {
					// No Secret Provided
					if hasFactorIndex {
						// Provided Two Factor Index
						// Send secret
						attempt := *newTwoFactorAttempt()
						h.twoFactorCheck[vUser] = attempt
						h.sendSecret(factorIndex, rxConfig, attempt)
						return c.JSON(fiber.Map{"t": h.twoFactorTimeoutSec})
					} else {
						// No Index Provided
						if numFactors == 1 {
							// Only One So Send
							// Send secret
							attempt := *newTwoFactorAttempt()
							h.twoFactorCheck[vUser] = attempt
							h.sendSecret(0, rxConfig, attempt)
							return c.JSON(fiber.Map{"t": h.twoFactorTimeoutSec})
						} else {
							// Send Two Factor Options
							maskedOptions := getTwoFactorMaskedOptions(rxConfig)
							return c.JSON(fiber.Map{"o": maskedOptions})
						}
					}
				}
			} else {
				// Basic Auth
				t, err := h.createToken(user, timeNow)
				if err != nil {
					h.loginLogger.Printf("%s,error,%s,%s,%s\r\n", getFormattedKitchenTimestamp(timeNow), vUser, c.IP(), c.IPs())
					return c.SendStatus(fiber.StatusInternalServerError)
				}
				// Success Basic Auth
				h.loginLogger.Printf("%s,success,%s,%s,%s\r\n", getFormattedKitchenTimestamp(timeNow), vUser, c.IP(), c.IPs())
				return c.JSON(fiber.Map{"c": t})
			}
		}
	}
	h.loginLogger.Printf("%s,unauthorized,,%s,%s\r\n", getFormattedKitchenTimestamp(timeNow), c.IP(), c.IPs())
	return c.SendStatus(fiber.StatusUnauthorized)
}

func (h *Http) loginMiddleware() func(*fiber.Ctx) error {
	return jwtware.New(jwtware.Config{
		SigningKey: []byte(h.loginKey),
		SuccessHandler: func(c *fiber.Ctx) error {
			h.accessLogger.Printf("%s,access,%s,%s\r\n", getFormattedKitchenTimestamp(time.Now()), c.IP(), c.IPs())
			return c.Next()
		},
		ErrorHandler: func(c *fiber.Ctx, e error) error {
			h.accessLogger.Printf("%s,%s,%s,%s\r\n", getFormattedKitchenTimestamp(time.Now()), e, c.IP(), c.IPs())
			return c.SendStatus(fiber.StatusUnauthorized)
		},
	})
}
