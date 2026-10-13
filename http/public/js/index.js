let hostname = $(location).attr('host');
let connected = false;
let webSockets = {};
let monitorPaused = {};
let activeMonitors = [];

$(document).ready(function () {
   $.ajaxSetup({
      beforeSend: function (xhr) {
         let jwt = Cookies.get("token");
         if (jwt) {
            xhr.setRequestHeader("Authorization", 'Bearer ' + jwt);
         }
      }
   });

   initGridControls();
   initStreamSettingsModal();
   heartbeat();
});

$(window).on("beforeunload", function () {
   connected = false;
   cleanupWebSockets();
});

const gridColLabels = {
   "auto": "Auto",
   "1": "1 Col",
   "2": "2 Cols",
   "3": "3 Cols",
   "4": "4 Cols"
};

// Grid column layout selection
function initGridControls() {
   let savedCols = localStorage.getItem("scout_grid_cols") || "auto";
   if (!["auto", "1", "2", "3", "4"].includes(String(savedCols))) {
      savedCols = "auto";
   }
   applyGridColumns(savedCols);

   $('.grid-dropdown-item').on('click', function () {
      let cols = $(this).data('cols');
      applyGridColumns(cols);
      localStorage.setItem("scout_grid_cols", cols);
   });
}

function applyGridColumns(cols) {
   let colsStr = String(cols);
   if (!["auto", "1", "2", "3", "4"].includes(colsStr)) {
      colsStr = "auto";
   }

   let container = $('#monitor-container');
   container.removeClass('cols-auto cols-1 cols-2 cols-3 cols-4');
   container.addClass(`cols-${colsStr}`);

   $('.grid-dropdown-item').removeClass('active');
   $(`.grid-dropdown-item[data-cols="${colsStr}"]`).addClass('active');

   let labelText = gridColLabels[colsStr] || "Auto";
   $('#grid-current-label').text(labelText);
}

// Heartbeat & Connection Loop
function heartbeat() {
   $.ajax({
      type: "GET",
      url: "heartbeat",
      timeout: 5000,
      error: function (request, error) {
         if (connected) {
            cleanup();
         }
         connected = false;
         if (request.status === 401) {
            authRequired = true;
            if (typeof updateAuthButtons === "function") {
               updateAuthButtons();
            }
            logout();
         }
      },
      success: function (data) {
         if (!connected) {
            connected = true;
            initialize();
         }
         return true;
      },
      complete: function () {
         setTimeout(heartbeat, 2000);
      }
   });
}

function initialize() {
   let pill = $('#status-pill');
   pill.removeClass('disconnected');
   $('#status').text("Connected");
   $('#memory-body').removeClass('d-none');
   getMonitorList();
   getMemory();
}

function cleanup() {
   cleanupWebSockets();
   let pill = $('#status-pill');
   pill.addClass('disconnected');
   $('#status').text("Disconnected");
   $('#memory').text("Offline");
   $('#monitor-container').empty();
   updateCamerasDropdown([]);
   activeMonitors = [];
}

// Monitor Discovery & Management
function getMonitorList() {
   if (!connected) return;
   $.ajax({
      type: "GET",
      url: "info/list",
      timeout: 5000,
      success: function (data) {
         if (data && data.NameList) {
            syncMonitors(data.NameList);
         }
      },
      complete: function () {
         if (connected) {
            setTimeout(getMonitorList, 5000);
         }
      }
   });
}

// Hidden cameras state persistence
function getHiddenCameras() {
   try {
      let raw = localStorage.getItem("scout_hidden_cameras");
      if (!raw) return [];
      let parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed : [];
   } catch (e) {
      return [];
   }
}

function setHiddenCameras(hiddenList) {
   try {
      localStorage.setItem("scout_hidden_cameras", JSON.stringify(hiddenList));
   } catch (e) {}
}

function isCameraHidden(name) {
   return getHiddenCameras().includes(name);
}

// Per-camera Stream Settings (Resolution & Quality)
const DEFAULT_STREAM_SETTINGS = { width: 640, quality: 50 };

function getAllStreamSettings() {
   try {
      let raw = localStorage.getItem("scout_camera_settings");
      if (!raw) return {};
      let parsed = JSON.parse(raw);
      return typeof parsed === 'object' && parsed !== null ? parsed : {};
   } catch (e) {
      return {};
   }
}

function getStreamSettings(name) {
   let all = getAllStreamSettings();
   let camSettings = all[name] || {};
   let width = parseInt(camSettings.width, 10);
   let quality = parseInt(camSettings.quality, 10);

   if (!width || isNaN(width) || width <= 0) width = DEFAULT_STREAM_SETTINGS.width;
   if (!quality || isNaN(quality) || quality <= 0) quality = DEFAULT_STREAM_SETTINGS.quality;

   return { width, quality };
}

function setStreamSettings(name, width, quality) {
   let all = getAllStreamSettings();
   all[name] = { width: parseInt(width, 10), quality: parseInt(quality, 10) };
   try {
      localStorage.setItem("scout_camera_settings", JSON.stringify(all));
   } catch (e) {}
}

function setAllStreamSettings(width, quality) {
   let all = getAllStreamSettings();
   activeMonitors.forEach(function (name) {
      all[name] = { width: parseInt(width, 10), quality: parseInt(quality, 10) };
   });
   try {
      localStorage.setItem("scout_camera_settings", JSON.stringify(all));
   } catch (e) {}
}

function formatResLabel(width) {
   switch (parseInt(width, 10)) {
      case 320: return "320p";
      case 480: return "480p";
      case 640: return "640p";
      case 960: return "960p";
      case 1280: return "720p";
      case 1920: return "1080p";
      default: return `${width}w`;
   }
}

let currentSettingsCamera = "";

function updateCameraSettingsUI(name) {
   let settings = getStreamSettings(name);
   let resLabel = formatResLabel(settings.width);
   let badgeText = `${resLabel} • ${settings.quality}%`;

   let badge = $(`#${name}-res-badge`);
   badge.text(badgeText);
   badge.attr('title', `Click to adjust resolution & quality (${settings.width}px @ ${settings.quality}%)`);

   if (currentSettingsCamera === name && $('#stream-settings-modal').hasClass('show')) {
      updateStreamSettingsModalUI(name);
   }
}

function restartCameraWebSocket(name) {
   if (monitorPaused[name]) return;
   if (webSockets[name]) {
      try {
         webSockets[name].onclose = null;
         webSockets[name].onerror = null;
         webSockets[name].close();
      } catch (e) {}
      delete webSockets[name];
   }
   createWebSocket(name);
}

function openStreamSettingsModal(name) {
   currentSettingsCamera = name || activeMonitors[0] || "";
   if (!currentSettingsCamera) return;

   // Populate camera selector dropdown
   let select = $('#modal-stream-camera-select');
   select.empty();
   activeMonitors.forEach(function (mon) {
      select.append(`<option value="${mon}" ${mon === currentSettingsCamera ? 'selected' : ''}>${mon}</option>`);
   });

   if (activeMonitors.length <= 1) {
      $('#modal-camera-select-group').addClass('d-none');
   } else {
      $('#modal-camera-select-group').removeClass('d-none');
   }

   updateStreamSettingsModalUI(currentSettingsCamera);
   $('#stream-settings-modal').modal('show');
}

function updateStreamSettingsModalUI(name) {
   currentSettingsCamera = name;
   $('#modal-stream-cam-name').text(name);
   $('#modal-stream-camera-select').val(name);

   let settings = getStreamSettings(name);

   // Update active pills in modal
   $('.modal-res-pills .stream-pill-btn').removeClass('active');
   $(`.modal-res-pills .stream-pill-btn[data-res="${settings.width}"]`).addClass('active');

   $('.modal-quality-pills .stream-pill-btn').removeClass('active');
   $(`.modal-quality-pills .stream-pill-btn[data-quality="${settings.quality}"]`).addClass('active');
}

function initStreamSettingsModal() {
   // Camera selector change inside modal
   $('#modal-stream-camera-select').off('change').on('change', function () {
      let selectedCam = $(this).val();
      updateStreamSettingsModalUI(selectedCam);
   });

   // Resolution pill click inside modal
   $('.modal-res-pills .stream-pill-btn').off('click').on('click', function () {
      let newRes = parseInt($(this).data('res'), 10);
      let cur = getStreamSettings(currentSettingsCamera);
      setStreamSettings(currentSettingsCamera, newRes, cur.quality);
      updateCameraSettingsUI(currentSettingsCamera);
      updateStreamSettingsModalUI(currentSettingsCamera);
      restartCameraWebSocket(currentSettingsCamera);
   });

   // Quality pill click inside modal
   $('.modal-quality-pills .stream-pill-btn').off('click').on('click', function () {
      let newQuality = parseInt($(this).data('quality'), 10);
      let cur = getStreamSettings(currentSettingsCamera);
      setStreamSettings(currentSettingsCamera, cur.width, newQuality);
      updateCameraSettingsUI(currentSettingsCamera);
      updateStreamSettingsModalUI(currentSettingsCamera);
      restartCameraWebSocket(currentSettingsCamera);
   });

   // Apply to all cameras button
   $('#modal-btn-apply-all').off('click').on('click', function () {
      let cur = getStreamSettings(currentSettingsCamera);
      setAllStreamSettings(cur.width, cur.quality);
      activeMonitors.forEach(function (mon) {
         updateCameraSettingsUI(mon);
         restartCameraWebSocket(mon);
      });
      let btn = $(this);
      let orig = btn.text();
      btn.text("Applied!").addClass('btn-info').removeClass('btn-outline-info');
      setTimeout(function () {
         btn.text(orig).addClass('btn-outline-info').removeClass('btn-info');
      }, 1200);
   });

   // Reset default button
   $('#modal-btn-reset-default').off('click').on('click', function () {
      setStreamSettings(currentSettingsCamera, DEFAULT_STREAM_SETTINGS.width, DEFAULT_STREAM_SETTINGS.quality);
      updateCameraSettingsUI(currentSettingsCamera);
      updateStreamSettingsModalUI(currentSettingsCamera);
      restartCameraWebSocket(currentSettingsCamera);
   });
}

function syncMonitors(newList) {
   let isSame = activeMonitors.length === newList.length &&
      activeMonitors.every((val, index) => val === newList[index]);

   let hidden = getHiddenCameras();
   let visibleCount = activeMonitors.filter(n => !hidden.includes(n)).length;

   if (isSame && ($('#monitor-container .camera-card').length === visibleCount)) {
      return;
   }

   activeMonitors = [...newList];
   renderMonitors(activeMonitors);
}

function updateCamerasDropdownState() {
   let hidden = getHiddenCameras();
   let visibleCount = 0;

   activeMonitors.forEach(function (name) {
      let isHidden = hidden.includes(name);
      if (!isHidden) visibleCount++;
      $(`.camera-checkbox[data-monitor="${name}"]`).prop('checked', !isHidden);
   });

   $('#cameras-badge').text(`${visibleCount}/${activeMonitors.length}`);
}

function updateCamerasDropdown(allMonitors) {
   let list = $('#cameras-checkboxes-list');
   list.empty();

   let hidden = getHiddenCameras();
   let visibleCount = 0;

   allMonitors.forEach(function (name) {
      let isHidden = hidden.includes(name);
      if (!isHidden) visibleCount++;

      let itemHtml = `
         <label class="camera-filter-item">
            <input type="checkbox" class="camera-checkbox mr-2" data-monitor="${name}" ${!isHidden ? 'checked' : ''}>
            <span>${name}</span>
         </label>
      `;
      list.append(itemHtml);
   });

   $('#cameras-badge').text(`${visibleCount}/${allMonitors.length}`);

   // Prevent closing dropdown when clicking inside checkbox list
   $('#cameras-checkboxes-list').off('click').on('click', function (e) {
      e.stopPropagation();
   });

   // Bind checkbox change
   $('.camera-checkbox').off('change').on('change', function () {
      let name = $(this).data('monitor');
      let isChecked = $(this).is(':checked');
      if (isChecked) {
         showCamera(name);
      } else {
         hideCamera(name);
      }
   });

   // Bind show all in dropdown
   $('#dropdown-show-all').off('click').on('click', function () {
      showAllCameras();
   });

   // Bind clear all in dropdown
   $('#dropdown-clear-all').off('click').on('click', function () {
      clearAllCameras();
   });
}

function renderSingleCameraCard(name) {
   let isPaused = monitorPaused[name] || false;
   let pauseIcon = isPaused ? '#icon-play' : '#icon-pause';
   let pauseTitle = isPaused ? 'Resume Stream' : 'Pause Stream';
   let pauseClass = isPaused ? 'btn-icon btn-pause is-paused' : 'btn-icon btn-pause';
   let settings = getStreamSettings(name);
   let resLabel = formatResLabel(settings.width);
   let badgeText = `${resLabel} • ${settings.quality}%`;

   let cardHtml = `
      <div class="camera-card" id="card-${name}">
         <div class="camera-header">
            <div class="camera-title-group">
               <h5 class="camera-name" title="${name}">${name}</h5>
            </div>
            <div class="camera-badges-group">
               <span class="badge-fps badge-stream-res" id="${name}-res-badge" data-monitor="${name}" title="Click to adjust resolution & quality (${settings.width}px @ ${settings.quality}%)">${badgeText}</span>
               <div class="camera-fps-group" id="${name}-fps-group">
                  <span class="badge-fps fps-idle" id="${name}-reader-fps">In: 0 FPS</span>
                  <span class="badge-fps fps-idle" id="${name}-process-fps">Out: 0 FPS</span>
               </div>
            </div>
            <div class="camera-actions">
               <button type="button" class="btn-icon btn-stream-settings" data-monitor="${name}" title="Stream Settings (Quality & Resolution)">
                  <svg><use href="#icon-settings"></use></svg>
               </button>
               <button type="button" class="${pauseClass}" data-monitor="${name}" title="${pauseTitle}">
                  <svg id="${name}-pause-icon"><use href="${pauseIcon}"></use></svg>
               </button>
               <button type="button" class="btn-icon btn-fullscreen" data-monitor="${name}" title="Toggle Fullscreen">
                  <svg><use href="#icon-fullscreen"></use></svg>
               </button>
               <button type="button" class="btn-icon btn-hide" data-monitor="${name}" title="Hide Camera">
                  <svg><use href="#icon-eye-off"></use></svg>
               </button>
            </div>
         </div>
         <div class="video-viewport" id="${name}-viewport">
            <div id="${name}-spinner" class="spinner-border text-info" style="width: 3.5rem; height: 3.5rem;" role="status">
               <span class="sr-only">Loading stream...</span>
            </div>
            <img id="${name}" class="feed-img d-none" alt="${name} Feed">
         </div>
      </div>
   `;

   let targetIdx = activeMonitors.indexOf(name);
   let inserted = false;
   if (targetIdx !== -1) {
      for (let i = targetIdx + 1; i < activeMonitors.length; i++) {
         let nextCard = $(`#card-${activeMonitors[i]}`);
         if (nextCard.length > 0) {
            $(cardHtml).insertBefore(nextCard);
            inserted = true;
            break;
         }
      }
   }
   if (!inserted) {
      $('#monitor-container').append(cardHtml);
   }
}

function renderAllHiddenState() {
   let container = $('#monitor-container');
   container.html(`
      <div class="empty-cameras" id="all-hidden-msg">
         <h4>All cameras are hidden</h4>
         <p>Use the Cameras filter menu to show cameras, or click below.</p>
         <button type="button" class="btn btn-sm btn-primary btn-show-all-cams mt-2">Show All Cameras</button>
      </div>
   `);
   $('.btn-show-all-cams').off('click').on('click', function () {
      showAllCameras();
   });
}

function renderMonitors(monitorNames) {
   let container = $('#monitor-container');
   cleanupWebSockets();
   container.empty();

   if (!monitorNames || monitorNames.length === 0) {
      container.html(`
         <div class="empty-cameras">
            <h4>No active cameras found</h4>
            <p>Ensure video sources are configured in .config/manage.yaml</p>
         </div>
      `);
      updateCamerasDropdown([]);
      return;
   }

   updateCamerasDropdown(monitorNames);

   let visibleMonitors = monitorNames.filter(name => !isCameraHidden(name));

   if (visibleMonitors.length === 0) {
      renderAllHiddenState();
      return;
   }

   visibleMonitors.forEach(function (name) {
      renderSingleCameraCard(name);
      createWebSocket(name);
      getMonitorInfo(name);
   });

   attachCardEventListeners();
}

function hideCamera(name) {
   let hidden = getHiddenCameras();
   if (!hidden.includes(name)) {
      hidden.push(name);
      setHiddenCameras(hidden);
   }

   if (webSockets[name]) {
      try {
         webSockets[name].close();
      } catch (e) {}
      delete webSockets[name];
   }

   $(`#card-${name}`).remove();
   updateCamerasDropdownState();

   let visibleCount = activeMonitors.filter(n => !getHiddenCameras().includes(n)).length;
   if (visibleCount === 0 && activeMonitors.length > 0) {
      renderAllHiddenState();
   }
}

function showCamera(name) {
   let hidden = getHiddenCameras();
   let idx = hidden.indexOf(name);
   if (idx !== -1) {
      hidden.splice(idx, 1);
      setHiddenCameras(hidden);
   }

   $('#all-hidden-msg').remove();

   if ($(`#card-${name}`).length === 0) {
      renderSingleCameraCard(name);
      createWebSocket(name);
      getMonitorInfo(name);
      attachCardEventListeners();
   }

   updateCamerasDropdownState();
}

function showAllCameras() {
   setHiddenCameras([]);
   renderMonitors(activeMonitors);
}

function clearAllCameras() {
   setHiddenCameras([...activeMonitors]);
   cleanupWebSockets();
   $('#monitor-container').empty();
   renderAllHiddenState();
   updateCamerasDropdownState();
}

function attachCardEventListeners() {
   // Settings Button and Badge -> opens Stream Settings Modal
   $('.btn-stream-settings, .badge-stream-res').off('click').on('click', function (e) {
      e.stopPropagation();
      let name = $(this).data('monitor');
      openStreamSettingsModal(name);
   });

   // Pause / Resume Button
   $('.btn-pause').off('click').on('click', function () {
      let name = $(this).data('monitor');
      togglePause(name);
   });

   // Fullscreen Button
   $('.btn-fullscreen').off('click').on('click', function () {
      let name = $(this).data('monitor');
      toggleFullscreen(name);
   });

   // Hide Camera Button
   $('.btn-hide').off('click').on('click', function () {
      let name = $(this).data('monitor');
      hideCamera(name);
   });

   // Show All Cameras buttons
   $('.btn-show-all-cams').off('click').on('click', function () {
      showAllCameras();
   });
}

function togglePause(name) {
   let isPaused = monitorPaused[name] || false;
   let btn = $(`.btn-pause[data-monitor="${name}"]`);
   if (isPaused) {
      // Resume
      monitorPaused[name] = false;
      btn.removeClass('is-paused');
      btn.attr('title', 'Pause Stream');
      $(`#${name}-pause-icon use`).attr('href', '#icon-pause');
      createWebSocket(name);
   } else {
      // Pause
      monitorPaused[name] = true;
      if (webSockets[name]) {
         webSockets[name].close();
         delete webSockets[name];
      }
      btn.addClass('is-paused');
      btn.attr('title', 'Resume Stream');
      $(`#${name}-pause-icon use`).attr('href', '#icon-play');
   }
}

function toggleFullscreen(name) {
   let card = document.getElementById(`card-${name}`);
   if (!card) return;

   if (!document.fullscreenElement) {
      if (card.requestFullscreen) {
         card.requestFullscreen();
      } else if (card.webkitRequestFullscreen) {
         card.webkitRequestFullscreen();
      }
   } else {
      if (document.exitFullscreen) {
         document.exitFullscreen();
      } else if (document.webkitExitFullscreen) {
         document.webkitExitFullscreen();
      }
   }
}

// Track native fullscreen change to update icon
document.addEventListener('fullscreenchange', function () {
   $('.btn-fullscreen use').attr('href', document.fullscreenElement ? '#icon-compress' : '#icon-fullscreen');
});

// WebSocket Streaming
function createWebSocket(monitorName) {
   if (monitorPaused[monitorName]) return null;

   if (webSockets[monitorName] && webSockets[monitorName].readyState === WebSocket.OPEN) {
      return webSockets[monitorName];
   }

   let settings = getStreamSettings(monitorName);
   let jwt = Cookies.get("token") || "";
   let wsPre = ('https:' === document.location.protocol ? 'wss' : 'ws');
   let url = `${wsPre}://${hostname}/live/${monitorName}?quality=${settings.quality}&width=${settings.width}&token=${jwt}`;

   let ws = new WebSocket(url);
   webSockets[monitorName] = ws;

   ws.onopen = function () {
      requestAnimationFrame(function () {
         let monImg = $(`#${monitorName}`);
         if (!monImg.attr('src')) {
            $(`#${monitorName}-spinner`).removeClass('d-none');
            monImg.addClass('d-none');
         }
      });
   };

   ws.onmessage = function (evt) {
      if (monitorPaused[monitorName]) return;
      let blobData = evt.data;
      blobbase64tostring(blobData, function (b64) {
         let binData = base64stringtobinaryuint8array(b64);
         let decoded = pako.inflate(binData);
         let decodedBlob = new Blob([decoded]);
         blobbase64tostring(decodedBlob, function (b64Decoded) {
            let srcValue = `data:image/jpg;base64,${b64Decoded}`;
            requestAnimationFrame(function () {
               let monImg = $(`#${monitorName}`);
               let monSpinner = $(`#${monitorName}-spinner`);
               monSpinner.addClass('d-none');
               monImg.removeClass('d-none');
               monImg.attr('src', srcValue);
            });
         });
      });
   };

   ws.onclose = function () {
      delete webSockets[monitorName];
      if (connected && !monitorPaused[monitorName]) {
         setTimeout(createWebSocket, 1500, monitorName);
      }
   };

   ws.onerror = function () {
      ws.close();
   };

   return ws;
}

function cleanupWebSockets() {
   Object.keys(webSockets).forEach(function (name) {
      if (webSockets[name]) {
         try {
            webSockets[name].close();
         } catch (e) {}
      }
   });
   webSockets = {};
}

// Stats & Metrics
function getMonitorInfo(monitorName) {
   if (!connected) return;
   $.ajax({
      type: "GET",
      url: `info/${monitorName}`,
      timeout: 3000,
      success: function (data) {
         if (data) {
            let inFps = data.ReaderInFps || 0;
            let outFps = data.ReaderOutFps || 0;

            let inBadge = $(`#${monitorName}-reader-fps`);
            let outBadge = $(`#${monitorName}-process-fps`);

            inBadge.text(`In: ${inFps} FPS`);
            outBadge.text(`Out: ${outFps} FPS`);

            inBadge.removeClass('fps-warn fps-idle');
            outBadge.removeClass('fps-warn fps-idle');

            if (inFps === 0) inBadge.addClass('fps-warn');
            if (outFps === 0) outBadge.addClass('fps-warn');
         }
      },
      complete: function () {
         if (connected && !monitorPaused[monitorName]) {
            setTimeout(getMonitorInfo, 2000, monitorName);
         }
      }
   });
}

function getMemory() {
   if (!connected) return;
   $.ajax({
      type: "GET",
      url: "memory",
      timeout: 3000,
      success: function (data) {
         if (data) {
            let memoryDiv = $('#memory');
            let appMB = data.RAMAppMB || 0;
            let sysMB = data.RAMSystemMB || 0;
            let percent = sysMB > 0 ? ((appMB / sysMB) * 100).toFixed(1) : 0;

            memoryDiv.text(`App: ${appMB} MB | Sys: ${(sysMB / 1024).toFixed(1)} GB (${percent}%)`);
         }
      },
      complete: function () {
         if (connected) {
            setTimeout(getMemory, 3000);
         }
      }
   });
}
