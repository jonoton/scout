$(document).ready(function () {
    let token = Cookies.get('token');
    if (token === undefined) {
        $("#logout-button").addClass('d-none');
        $("#login-button").removeClass('d-none');
    } else {
        $("#logout-button").removeClass('d-none');
        $("#login-button").addClass('d-none');
    }

    $(document).bind('keypress', function (e) {
        if ($("#login").is(":visible") && e.keyCode == 13) {
            $('#login').trigger('click');
        }
    });

    $("#login").click(function () {
        let user = $("#login_user").val().trim();
        let pass = $("#login_pwd").val().trim();
        $("#bad_user").addClass('d-none');
        if (user == "" || pass == "") {
            return;
        }
        let uh = CryptoJS.MD5(user).toString();
        let ph = CryptoJS.MD5(pass).toString();
        $.post("login", { a: uh, b: ph }, function (data) {
            Cookies.set('token', data.c)
            $("#bad_user").addClass('d-none');
            $("#logout-button").removeClass('d-none');
            $("#login-button").addClass('d-none');
            $('#login-modal').modal('toggle');
        }).fail(function () {
            $("#bad_user").removeClass('d-none');
        });
    });
    
    $("#logout-button").click(function () {
        logout()
    });

    $('#login-modal').on('shown.bs.modal', function () {
        $("#login_user").focus();
    });
});

function logout() {
    Cookies.remove('token');
    $("#logout-button").addClass('d-none');
    $("#login-button").removeClass('d-none');
}
