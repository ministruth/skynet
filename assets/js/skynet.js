toastr.options.newestOnTop = true;
toastr.options.progressBar = true;
toastr.options.timeOut = "3000";
toastr.options.preventDuplicates = true;

function JSONAction(method, url, d) {
  let csrfToken = document.getElementsByName("csrf-token")[0].content;
  return $.ajax(url, {
    type: method,
    data: JSON.stringify(d),
    contentType: "application/json",
    headers: { "X-CSRF-Token": csrfToken },
  }).fail(function (d) {
    if (d.responseText == undefined) toastr.error("Connect error");
    else if (d.responseText == "") toastr.error(d.statusText);
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

function InsertParam(key, value) {
  key = encodeURIComponent(key);
  value = encodeURIComponent(value);

  var kvp = document.location.search.substr(1).split("&");
  let i = 0;

  for (; i < kvp.length; i++) {
    if (kvp[i].startsWith(key + "=")) {
      let pair = kvp[i].split("=");
      pair[1] = value;
      kvp[i] = pair.join("=");
      break;
    }
  }

  if (i >= kvp.length) {
    kvp[kvp.length] = [key, value].join("=");
  }

  let params = kvp.join("&");
  document.location.search = params;
}

function GetPageSize(s, sl) {
  var urlParams = new URLSearchParams(window.location.search);
  let size = parseInt(urlParams.get("size"));
  let page = parseInt(urlParams.get("page"));
  if (size == null || isNaN(size) || !sl.includes(size)) {
    size = s;
    $(".pagebtn").val(s);
  } else {
    $(".pagebtn").val(size);
  }
  if (page == null || isNaN(page) || page <= 0) page = 1;
  return [page, size];
}

Date.prototype.Format = function (fmt) {
  //author: meizz
  var o = {
    "M+": this.getMonth() + 1,
    "d+": this.getDate(),
    "h+": this.getHours(),
    "m+": this.getMinutes(),
    "s+": this.getSeconds(),
    "q+": Math.floor((this.getMonth() + 3) / 3),
    S: this.getMilliseconds(),
  };
  if (/(y+)/.test(fmt))
    fmt = fmt.replace(
      RegExp.$1,
      (this.getFullYear() + "").substr(4 - RegExp.$1.length)
    );
  for (var k in o)
    if (new RegExp("(" + k + ")").test(fmt))
      fmt = fmt.replace(
        RegExp.$1,
        RegExp.$1.length == 1 ? o[k] : ("00" + o[k]).substr(("" + o[k]).length)
      );
  return fmt;
};

function updateProgress() {
  $("[role='progressbar']").each((_, e) => {
    e.style.width = e.getAttribute("aria-valuenow") + "%";
  });
}
