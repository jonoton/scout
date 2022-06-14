$(document).ready(function () {
  let token = Cookies.get("token");
  if (token === undefined) {
    $("#logout-button").addClass("d-none");
    $("#login-button").removeClass("d-none");
  } else {
    $("#logout-button").removeClass("d-none");
    $("#login-button").addClass("d-none");
  }

  $("#login_user").keypress(function (e) {
    if (e.which == 13) {
      $("#login-btn").trigger("click");
      return false;
    }
  });
  $("#login_pwd").keypress(function (e) {
    if (e.which == 13) {
      $("#login-btn").trigger("click");
      return false;
    }
  });
  $("#login_passcode").keypress(function (e) {
    if (e.which == 13) {
      $("#login-passcode-btn").trigger("click");
      return false;
    }
  });
  $("#login-modal").modal({ show: false, backdrop: "static" });
  $("#login-choose-modal").modal({ show: false, backdrop: "static" });
  $("#login-passcode-modal").modal({ show: false, backdrop: "static" });
  $("#login-btn").click(function () {
    let user = $("#login_user").val().trim();
    let pass = $("#login_pwd").val().trim();
    $("#bad_user").addClass("d-none");
    if (user == "" || pass == "") {
      return;
    }
    let uh = CryptoJS.SHA256(user).toString();
    let ph = CryptoJS.SHA256(pass).toString();
    $.post("login", { a: uh, b: ph }, function (data) {
      $("#bad_user").addClass("d-none");
      $("#login-modal").modal("toggle");
      if (data.c) {
        acceptLogin(data.c);
      } else if (data.o) {
        $("#login_choose").empty();
        $.each(data.o, function (i, item) {
          $("#login_choose").append(
            $("<option>", {
              value: i,
              text: item,
            })
          );
        });
        $("#login-choose-modal").modal("show");
      } else if (data.t) {
        setupPasscodeModal(data.t);
        $("#login-passcode-modal").modal("show");
      }
    }).fail(function () {
      $("#bad_user").removeClass("d-none");
    });
  });

  $("#login-choose-btn").click(function () {
    let user = $("#login_user").val().trim();
    let pass = $("#login_pwd").val().trim();
    if (user == "" || pass == "") {
      return;
    }
    let uh = CryptoJS.SHA256(user).toString();
    let ph = CryptoJS.SHA256(pass).toString();
    let y = $("#login_choose").val();
    $.post("login", { a: uh, b: ph, y: y }, function (data) {
      $("#login-choose-modal").modal("toggle");
      if (data.t) {
        setupPasscodeModal(data.t);
        $("#login-passcode-modal").modal("show");
      }
    }).fail(function () {});
  });

  $("#login-passcode-btn").click(function () {
    let user = $("#login_user").val().trim();
    let pass = $("#login_pwd").val().trim();
    $("#bad_passcode").addClass("d-none");
    if (user == "" || pass == "") {
      return;
    }
    let uh = CryptoJS.SHA256(user).toString();
    let ph = CryptoJS.SHA256(pass).toString();
    $.post(
      "login",
      { a: uh, b: ph, z: $("#login_passcode").val().trim() },
      function (data) {
        $("#login-passcode-modal").modal("toggle");
        if (data.c) {
          acceptLogin(data.c);
        }
      }
    ).fail(function () {
      $("#login-passcode-btn").prop("disabled", true);
      $("#bad_passcode").removeClass("d-none");
    });
  });

  $("#logout-button").click(function () {
    logout();
  });

  $("#login-modal").on("shown.bs.modal", function () {
    $("#login_user").focus();
  });
  $("#login-choose-modal").on("shown.bs.modal", function () {
    $("#login_choose").focus();
  });
  $("#login-passcode-modal").on("shown.bs.modal", function () {
    $("#login_passcode").focus();
  });
});

function setupPasscodeModal(t) {
  $("#login_passcode").val("");
  $("#login-passcode-duration").html("Enter Passcode within " + t + " seconds");
  $("#bad_passcode").addClass("d-none");
  $("#login-passcode-btn").prop("disabled", false);
}

function acceptLogin(c) {
  Cookies.set("token", c);
  $("#logout-button").removeClass("d-none");
  $("#login-button").addClass("d-none");
}

function logout() {
  Cookies.remove("token");
  $("#logout-button").addClass("d-none");
  $("#login-button").removeClass("d-none");
}
