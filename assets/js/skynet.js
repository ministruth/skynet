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
    else if (d.responseText == "") toastr.error(d.statusText);
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

function SplitPage(size, tot) {
  if (tot == 0) return 1;
  if (tot % size == 0) return tot / size;
  return parseInt(tot / size) + 1;
}

function ParsePageItem(page, totPage) {
  let ret = [];
  ret = ret.concat("«");
  if (totPage <= 5) {
    for (let i = 1; i <= totPage; i++) ret = ret.concat(i.toString());
  } else {
    let low = parseInt(Math.max(page - 2, 1));
    let high = parseInt(Math.min(page + 2, totPage));
    if (low == 1) {
      ret = ret.concat(["1", "2", "3", "4", "...", totPage.toString()]);
    } else if (high == totPage) {
      ret = ret.concat(["1", "..."]);
      for (let i = totPage - 3; i <= totPage; i++)
        ret = ret.concat(i.toString());
    } else {
      ret = ret.concat(["1", "..."]);
      for (let i = page - 1; i <= page + 1; i++) ret = ret.concat(i.toString());
      ret = ret.concat(["...", totPage.toString()]);
    }
  }
  ret = ret.concat("»");
  return ret;
}

let _page_status = {};

function ParsePage(pid, defSize, defSizeList, dataFunc, loopTime) {
  let tot = parseInt($(`ul[data-page="${pid}"]`).attr("data-total"));
  let url = ParseURLParam(pid, defSize, defSizeList);
  let size = url[1];
  let totPage = SplitPage(size, tot);
  let page = Math.min(url[0], totPage);
  _page_status[pid] = {
    size: size,
    page: page,
    totPage: totPage,
    dirty: true,
    data: "",
  };
  RenderPage(pid, dataFunc);
  if (loopTime != 0)
    setInterval(() => {
      RenderPage(pid, dataFunc);
    }, loopTime);
}

function UpdateTotal(pid, tot, dataFunc) {
  if (tot != $(`ul[data-page="${pid}"]`).attr("data-total")) {
    $(`ul[data-page="${pid}"]`).attr("data-total", tot.toString());
    let totPage = SplitPage(_page_status[pid].size, tot);
    _page_status[pid].page = Math.min(_page_status[pid].page, totPage);
    _page_status[pid].totPage = totPage;
    _page_status[pid].dirty = true;
    RenderPage(pid, dataFunc);
    return true;
  }
  return false;
}

function RenderPage(pid, dataFunc) {
  if (_page_status[pid].dirty) {
    _page_status[pid].dirty = false;
    let ul = $(`ul[data-page="${pid}"]`);
    ul.empty();
    let item = ParsePageItem(_page_status[pid].page, _page_status[pid].totPage);
    item.forEach((i) => {
      let str = "";
      if (i == "«") {
        if (_page_status[pid].page == 1)
          str += `<li class="page-item disabled"><a class="page-link" data-page="${pid}">${i}</a></li>`;
        else
          str += `<li class="page-item"><a class="page-link" data-page="${pid}" data-pageNumber="${
            _page_status[pid].page - 1
          }">${i}</a></li>`;
      } else if (i == "»") {
        if (_page_status[pid].page == _page_status[pid].totPage)
          str += `<li class="page-item disabled"><a class="page-link" data-page="${pid}">${i}</a></li>`;
        else
          str += `<li class="page-item"><a class="page-link" data-page="${pid}" data-pageNumber="${
            _page_status[pid].page + 1
          }">${i}</a></li>`;
      } else if (i == _page_status[pid].page) {
        str += `<li class="page-item active"><a class="page-link" data-page="${pid}" data-pageNumber="${i}">${i}</a></li>`;
      } else if (i == "...") {
        str += `<li class="page-item disabled"><a class="page-link" data-page="${pid}">${i}</a></li>`;
      } else {
        str += `<li class="page-item"><a class="page-link" data-page="${pid}" data-pageNumber="${i}">${i}</a></li>`;
      }
      ul.append(str);
    });
    ul.append(
      `<li>
          <select class="form-control pagebtn" data-page="${pid}">
            <option value="5">5 / page</option>
            <option value="10">10 / page</option>
            <option value="20">20 / page</option>
            <option value="50">50 / page</option>
          </select>
      </li>`
    );
    $(`select[data-page="${pid}"]`).val(_page_status[pid].size.toString());
    $(`.page-link[data-page="${pid}"]`).click((e) => {
      _page_status[pid].dirty = true;
      _page_status[pid].page = parseInt(
        e.target.getAttribute("data-pageNumber")
      );
      RenderPage(pid, dataFunc);
    });
    $(`select[data-page="${pid}"]`).on("change", function () {
      let tot = parseInt($(`ul[data-page="${pid}"]`).attr("data-total"));
      let totPage = SplitPage(parseInt(this.value), tot);
      _page_status[pid].page = Math.min(_page_status[pid].page, totPage);
      _page_status[pid].totPage = totPage;
      _page_status[pid].size = parseInt(this.value);
      _page_status[pid].dirty = true;
      RenderPage(pid, dataFunc);
    });
  }
  dataFunc(_page_status[pid], _page_status[pid].page, _page_status[pid].size);
}

function ParseURLParam(pid, defSize, defSizeList) {
  var urlParams = new URLSearchParams(window.location.search);
  let size = parseInt(urlParams.get("size"));
  let page = parseInt(urlParams.get("page"));
  if (size == null || isNaN(size) || !defSizeList.includes(size)) {
    size = defSize;
    $(`select[data-page="${pid}"]`).val(defSize);
  } else {
    $(`select[data-page="${pid}"]`).val(size);
  }
  if (page == null || isNaN(page) || page <= 0) page = 1;
  return [page, size];
}

Date.prototype.TimeSince = function () {
  var seconds = Math.floor((new Date() - this) / 1000);
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
};

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
