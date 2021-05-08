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
    if (d.responseText == undefined) toastr.error("Connect error");
    else toastr.error(d.responseText);
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
    if (d.responseText == undefined) toastr.error("Connect error");
    else toastr.error(d.responseText);
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

function TimeSince(date) {
  var seconds = Math.floor((new Date() - date) / 1000);
  var interval = Math.floor(seconds / 31536000);

  if (interval > 1) return interval + " years";
  interval = Math.floor(seconds / 2592000);
  if (interval > 1) return interval + " months";
  interval = Math.floor(seconds / 86400);
  if (interval > 1) return interval + " days";
  interval = Math.floor(seconds / 3600);
  if (interval > 1) return interval + " hours";
  interval = Math.floor(seconds / 60);
  if (interval > 1) return interval + " minutes";
  if (Math.floor(seconds) >= 5) return Math.floor(seconds) + " seconds";
  else return "Just now";
}

function SizeString(bytes, decimals = 2) {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
}
