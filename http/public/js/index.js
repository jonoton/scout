let hostname = $(location).attr('host');
let connected = false;
let webSockets = [];

$(document).ready(function () {
   $.ajaxSetup({
      beforeSend: function (xhr) {
         let jwt = Cookies.get("token");
         xhr.setRequestHeader("Authorization", 'Bearer ' + jwt);
      }
   });
   heartbeat();
});

$(window).on("beforeunload", function () {
   connected = false;
   cleanupWebSockets();
})

function heartbeat() {
   $.ajax({
      type: "GET",
      url: "heartbeat",
      timeout: 0,
      error: function (request, error) {
         if (connected) {
            cleanup();
         }
         connected = false;
         // Unauthorized
         if (request.status == 401) {
            logout();
         }
      },
      success: function (data, textStatus, XMLHttpRequest) {
         if (!connected) {
            connected = true;
            initialize();
         }
         return true;
      },
      complete: function (jqXHR, textStatus) {
         setTimeout(heartbeat, 2000);
      }
   });
}

function getMonitorList() {
   $.ajax({
      type: "GET",
      url: "info/list",
      timeout: 0,
      error: function (request, error) {
         if (connected) {
            setTimeout(getMonitorList, 2000);
         }
      },
      success: function (data, textStatus, XMLHttpRequest) {
         if (data != null) {
            initMonitors(data.NameList);
         }
         return true;
      }
   });
}

function initialize() {
   $('#status-body').removeClass('bg-warning');
   $('#status-body').addClass('bgColorDark');
   $('#status').html("Connected");
   $('#memory-body').removeClass('d-none');
   getMonitorList();
   getMemory();
}

function initMonitors(monitorNames) {
   if (!("WebSocket" in window)) {
      return;
   }

   $('#monitor-container').empty();
   Object.keys(monitorNames).forEach(function (key) {
      let monitorName = monitorNames[key];
      let html = "";
      html += `
            <div class="row m-2 h-25">
               <div class="col-3 bg-light">
                  <div id="${monitorName}-info"><h5>${monitorName}</h5></div>
               </div>
               <div class="col-9 bg-dark text-center">
                  <div id="${monitorName}-div" class="d-none h-100 mw-100">
                     <div class="spinner-border text-info m-5" style="width: 5rem; height: 5rem;" role="status">
                        <span class="sr-only">Loading...</span>
                     </div>
                  </div>
                  <img id="${monitorName}" class="d-none h-100 mw-100">
               </div>
            </div>
      `;
      $('#monitor-container').append(html);
      let ws = createWebSocket(monitorName);
      webSockets.push(ws);
      getMonitorInfo(monitorName);
   });
}

function createWebSocket(monitorName) {
   let jwt = Cookies.get("token");
   let wsPre = ('https:' == document.location.protocol ? 'wss' : 'ws');
   let ws = new WebSocket(`${wsPre}://${hostname}/live/${monitorName}?quality=50&token=${jwt}`);
   ws.onopen = function () {
      requestAnimationFrame(function () {
         let monDiv = $(`#${monitorName}-div`);
         let monImg = $(`#${monitorName}`);
         monImg.addClass('d-none');
         monDiv.removeClass('d-none');
      });
   };
   ws.onmessage = function (evt) {
      let blobData = evt.data;
      blobbase64tostring(blobData, function (b64) {
         let binData = base64stringtobinaryuint8array(b64);
         let decoded = pako.inflate(binData);
         let decodedBlob = new Blob([decoded]);
         blobbase64tostring(decodedBlob, function (b64Decoded) {
            let srcValue = `data:image/jpg;base64,${b64Decoded}`;
            requestAnimationFrame(function () {
               let monDiv = $(`#${monitorName}-div`);
               let monImg = $(`#${monitorName}`);
               monDiv.addClass('d-none');
               monImg.removeClass('d-none');
               monImg.attr('src', `${srcValue}`);
            });
         });
      });
   };
   ws.onclose = function () {
      if (connected) {
         setTimeout(createWebSocket, 1000, monitorName);
      }
   };
   return ws;
}

function cleanup() {
   cleanupWebSockets();
   $('#status-body').removeClass('bgColorDark');
   $('#status-body').addClass('bg-warning');
   $('#status').html("Disconnected");
   $('#monitor-container').empty();
   $('#memory-body').addClass('d-none');
}

function cleanupWebSockets() {
   for (let i = 0; i < webSockets.length; i++) {
      let ws = webSockets[i];
      if (ws.readyState == WebSocket.OPEN) {
         ws.close();
      }
   }
   webSockets = [];
}

function getMonitorInfo(monitorName) {
   $.ajax({
      type: "GET",
      url: `info/${monitorName}`,
      timeout: 0,
      success: function (data, textStatus, XMLHttpRequest) {
         if (data != null) {
            let readerInFpsClass = "bg-success";
            if (data.ReaderInFps == 0) {
               readerInFpsClass = "bg-warning";
            }
            let readerOutFpsClass = "bg-success";
            if (data.ReaderOutFps == 0) {
               readerOutFpsClass = "bg-warning";
            }
            let html = `
               <h5>${data.Name}</h5>
               <div class='row'>
                  <div class='col text-white text-center ${readerInFpsClass}'>Reader ${data.ReaderInFps} FPS</div>
               </div>
               <div class='row'>
                  <div class='col text-white text-center ${readerOutFpsClass}'>Process ${data.ReaderOutFps} FPS</div>
               </div>
            `;
            $(`#${monitorName}-info`).html(html);
         }
         return true;
      },
      complete: function () {
         if (connected) {
            setTimeout(getMonitorInfo, 2000, monitorName);
         }
      }
   });
}

function getMemory() {
   $.ajax({
      type: "GET",
      url: "memory",
      timeout: 0,
      success: function (data, textStatus, XMLHttpRequest) {
         if (data != null) {
            let memoryDiv = $('#memory');
            memoryDiv.html(`RAM ${data.RAMAppMB} / ${data.RAMSystemMB} MB`);
            let percentUsage = 0;
            if (data.RAMSystemMB > 0) {
               percentUsage = (data.RAMAppMB / data.RAMSystemMB) * 100;
            }
            if (percentUsage <= 80) {
               memoryDiv.removeClass('bg-warning');
               memoryDiv.addClass('bgColorDark');
            } else {
               memoryDiv.removeClass('bgColorDark');
               memoryDiv.addClass('bg-warning');
            }
         }
         return true;
      },
      complete: function () {
         if (connected) {
            setTimeout(getMemory, 2000);
         }
      }
   });
}
