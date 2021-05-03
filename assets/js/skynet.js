toastr.options.newestOnTop = true;
toastr.options.progressBar = true;
toastr.options.timeOut = "3000";

function JSONAction(method, url, d) {
  let csrfToken = document.getElementsByName("csrf-token")[0].content;
  return $.ajax(url, {
    type: method,
    data: JSON.stringify(d),
    contentType: "application/json",
    headers: { "X-CSRF-Token": csrfToken },
  }).fail(function (d) {
    toastr.error(d.responseText);
  });
}

function GetData(form) {
  let data = {};
  new FormData(form).forEach((value, key) => (data[key] = value));
  return data;
}

function GetUrl(form) {
  return form.attributes["action"].value;
}

function JSONPost(url, d) {
  return JSONAction("POST", url, d);
}

function JSONDelete(url, d) {
  return JSONAction("DELETE", url, d);
}

function JSONPatch(url, d) {
  return JSONAction("PATCH", url, d);
}

function JSONGet(url) {
  return $.get(url).fail(function (d) {
    toastr.error(d.responseText);
  });
}

function DelayReload(t = 1000) {
  return function (d) {
    if (d.code != 0) toastr.error(d.msg);
    else
      toastr.success(d.msg, "", {
        timeOut: t,
        onHidden: () => {
          location.reload();
        },
      });
  };
}
