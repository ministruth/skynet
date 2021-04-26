toastr.options.newestOnTop = true;
toastr.options.progressBar = true;
toastr.options.timeOut = "3000";

function JSONPost(form, datacb) {
    let data = {};
    let csrfToken = document.getElementsByName("csrf-token")[0].content
    new FormData(form).forEach((value, key) => data[key] = value);
    data = datacb(data);
    return $.ajax(form.attributes["action"].value, {
        type: "POST",
        data: JSON.stringify(data),
        contentType: "application/json",
        headers: { "X-CSRF-Token": csrfToken }
    }).fail(function (d) {
        toastr.error(d.responseText);
    })
}