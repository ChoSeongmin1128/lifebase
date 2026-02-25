/**
 * common-ui.js
 * LifeBase 프로토타입 공통 UI 인터랙션
 *
 * 사용법: 각 프로토타입 HTML에서 <script src="shared/common-ui.js"></script> 로 로드한 뒤
 *         DOMContentLoaded 또는 인라인 <script> 에서 필요한 init 함수를 호출한다.
 */

/* ============================================================
 * 1. Sidebar toggle
 * ============================================================ */

function initSidebar() {
  var sidebar = document.getElementById('sidebar');
  var sidebarToggle = document.getElementById('sidebarToggle');
  if (!sidebar || !sidebarToggle) return;

  var collapsed = localStorage.getItem('lifebase-sidebar') === 'collapsed';
  if (collapsed) sidebar.classList.add('collapsed');

  sidebarToggle.addEventListener('click', function () {
    sidebar.classList.toggle('collapsed');
    localStorage.setItem(
      'lifebase-sidebar',
      sidebar.classList.contains('collapsed') ? 'collapsed' : 'expanded'
    );
  });
}

/* ============================================================
 * 2. Tab switching
 * ============================================================ */

/**
 * switchTab - 사이드바 탭 전환 핵심 로직
 * @param {string} tabName - 활성화할 탭 이름 (cloud, calendar, todo, settings 등)
 * @param {object} [opts] - 옵션
 * @param {boolean} [opts.wasActive] - 이미 활성 상태였는지 여부 (Cloud subnav 토글용)
 */
function switchTab(tabName, opts) {
  var wasActive = (opts && opts.wasActive) || false;
  var tabs = document.querySelectorAll('.nav-tab');
  var panels = document.querySelectorAll('.panel');
  var cloudSubnav = document.getElementById('cloudSubnav');

  tabs.forEach(function (b) { b.classList.remove('active'); });
  panels.forEach(function (p) { p.classList.remove('active'); });

  var targetTab = document.querySelector('.nav-tab[data-tab="' + tabName + '"]');
  if (targetTab) targetTab.classList.add('active');

  var targetPanel = document.getElementById('tab-' + tabName);
  if (targetPanel) targetPanel.classList.add('active');

  // Cloud subnav: toggle on re-click, closed on first click from other tab, close on other tabs
  if (cloudSubnav) {
    if (tabName === 'cloud') {
      if (wasActive) {
        cloudSubnav.classList.toggle('open');
      } else {
        cloudSubnav.classList.remove('open');
      }
    } else {
      cloudSubnav.classList.remove('open');
    }
  }

  // Calendar 뷰 전환 시 타임라인 행 높이 동기화
  if (tabName === 'calendar' && typeof syncYearTimelineRowHeight === 'function') {
    requestAnimationFrame(syncYearTimelineRowHeight);
  }
}

/**
 * initTabSwitching - 사이드바 .nav-tab 클릭 이벤트 바인딩
 */
function initTabSwitching() {
  var tabs = document.querySelectorAll('.nav-tab');

  tabs.forEach(function (button) {
    button.addEventListener('click', function () {
      var wasActive = button.classList.contains('active');
      switchTab(button.dataset.tab, { wasActive: wasActive });
    });
  });

  // Cloud subnav chevron 회전 동기화
  var cloudSubnav = document.getElementById('cloudSubnav');
  var cloudChevron = document.querySelector('[data-tab="cloud"] .nav-chevron');
  if (cloudSubnav && cloudChevron) {
    function syncChevron() {
      cloudChevron.style.transform = cloudSubnav.classList.contains('open')
        ? 'rotate(90deg)'
        : '';
    }
    syncChevron();
    new MutationObserver(syncChevron).observe(cloudSubnav, {
      attributes: true,
      attributeFilter: ['class']
    });
  }
}

/* ============================================================
 * 3. Theme toggle
 * ============================================================ */

/**
 * setTheme - 테마 적용
 * @param {string} mode - 'light' | 'dark' | 'system'
 */
function setTheme(mode) {
  var webThemeStatus = document.getElementById('webThemeStatus');
  var themeButtons = document.querySelectorAll('.theme-btn');

  if (mode === 'dark') {
    document.body.setAttribute('data-theme', 'dark');
    if (webThemeStatus) webThemeStatus.textContent = '테마: 다크';
  } else if (mode === 'light') {
    document.body.setAttribute('data-theme', 'light');
    if (webThemeStatus) webThemeStatus.textContent = '테마: 라이트';
  } else {
    document.body.removeAttribute('data-theme');
    if (webThemeStatus) webThemeStatus.textContent = '테마: 시스템';
  }

  themeButtons.forEach(function (btn) {
    btn.classList.toggle('active', btn.dataset.themeSet === mode);
  });
}

/**
 * initTheme - 저장된 테마 복원 및 버튼 이벤트 바인딩
 */
function initTheme() {
  var savedTheme = localStorage.getItem('lifebase-theme') || 'system';
  setTheme(savedTheme);

  var themeButtons = document.querySelectorAll('.theme-btn');
  themeButtons.forEach(function (button) {
    button.addEventListener('click', function () {
      var mode = button.dataset.themeSet;
      localStorage.setItem('lifebase-theme', mode);
      setTheme(mode);
    });
  });

  // 시스템 테마 변경 감지
  if (window.matchMedia) {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function () {
      var current = localStorage.getItem('lifebase-theme') || 'system';
      if (current === 'system') {
        setTheme('system');
      }
    });
  }
}

/* ============================================================
 * 4. Cloud dropdowns (Action / Create / Sort / View toggle)
 * ============================================================ */

/**
 * closeAllDropdowns - 열려 있는 모든 드롭다운 메뉴 닫기
 */
function closeAllDropdowns() {
  document.querySelectorAll(
    '.action-menu.open, .create-menu.open, .sort-menu.open'
  ).forEach(function (m) { m.classList.remove('open'); });
}

/**
 * initCloudDropdowns - Action/Create/Sort 메뉴, View 토글 이벤트 바인딩
 */
function initCloudDropdowns() {
  // --- Global click: action-toggle & outside-click close ---
  document.addEventListener('click', function (e) {
    var toggle = e.target.closest('[data-action-toggle]');
    if (toggle) {
      e.stopPropagation();
      var menu = toggle.nextElementSibling;
      var wasOpen = menu.classList.contains('open');
      closeAllDropdowns();
      if (!wasOpen) menu.classList.add('open');
      return;
    }
    if (
      !e.target.closest('.action-menu') &&
      !e.target.closest('.create-menu') &&
      !e.target.closest('.sort-menu')
    ) {
      closeAllDropdowns();
    }
  });

  // --- Cloud view switch (grid/list 등) ---
  var cloudViewSwitch = document.getElementById('cloudViewSwitch');
  if (cloudViewSwitch) {
    cloudViewSwitch.querySelectorAll('button').forEach(function (btn) {
      btn.addEventListener('click', function () {
        cloudViewSwitch.querySelectorAll('button').forEach(function (b) {
          b.classList.remove('active');
        });
        btn.classList.add('active');
      });
    });
  }

  // --- Create menu dropdown ---
  var createMenuBtn = document.getElementById('createMenuBtn');
  var createMenu = document.getElementById('createMenu');
  var newFileModal = document.getElementById('newFileModal');
  var newFileCancel = document.getElementById('newFileCancel');
  var newFileCreate = document.getElementById('newFileCreate');
  var newFileName = document.getElementById('newFileName');

  if (createMenuBtn && createMenu) {
    createMenuBtn.addEventListener('click', function (e) {
      e.stopPropagation();
      var wasOpen = createMenu.classList.contains('open');
      closeAllDropdowns();
      if (!wasOpen) createMenu.classList.add('open');
    });

    createMenu.querySelectorAll('.create-menu-item').forEach(function (item) {
      item.addEventListener('click', function () {
        createMenu.classList.remove('open');
        var type = item.dataset.create;
        if ((type === 'md' || type === 'txt') && newFileModal && newFileName) {
          newFileName.value = '';
          newFileModal.querySelectorAll('[data-filetype]').forEach(function (b) {
            b.classList.remove('active');
          });
          var target = newFileModal.querySelector('[data-filetype="' + type + '"]');
          if (target) target.classList.add('active');
          newFileModal.classList.add('open');
          setTimeout(function () { newFileName.focus(); }, 100);
        }
      });
    });
  }

  if (newFileCancel && newFileModal) {
    newFileCancel.addEventListener('click', function () {
      newFileModal.classList.remove('open');
    });
  }

  if (newFileModal) {
    newFileModal.addEventListener('click', function (e) {
      if (e.target === newFileModal) newFileModal.classList.remove('open');
    });

    newFileModal.querySelectorAll('[data-filetype]').forEach(function (btn) {
      btn.addEventListener('click', function () {
        newFileModal.querySelectorAll('[data-filetype]').forEach(function (b) {
          b.classList.remove('active');
        });
        btn.classList.add('active');
      });
    });
  }

  if (newFileCreate && newFileName && newFileModal) {
    newFileCreate.addEventListener('click', function () {
      var name = newFileName.value.trim();
      if (!name) { newFileName.focus(); return; }
      newFileModal.classList.remove('open');
    });
  }

  // --- Sort menu dropdown ---
  var sortMenuBtn = document.getElementById('sortMenuBtn');
  var sortMenu = document.getElementById('sortMenu');

  if (sortMenuBtn && sortMenu) {
    sortMenuBtn.addEventListener('click', function (e) {
      e.stopPropagation();
      var wasOpen = sortMenu.classList.contains('open');
      closeAllDropdowns();
      if (!wasOpen) sortMenu.classList.add('open');
    });

    sortMenu.querySelectorAll('.sort-menu-item').forEach(function (item) {
      item.addEventListener('click', function (e) {
        e.stopPropagation();
        var sortKey = item.dataset.sort;
        var sortDir = item.dataset.dir;
        if (sortKey) {
          sortMenu.querySelectorAll('[data-sort]').forEach(function (b) {
            b.classList.remove('active');
          });
          item.classList.add('active');
        }
        if (sortDir) {
          sortMenu.querySelectorAll('[data-dir]').forEach(function (b) {
            b.classList.remove('active');
          });
          item.classList.add('active');
        }
      });
    });
  }
}

/* ============================================================
 * 5. File table bulk selection
 * ============================================================ */

function initBulkSelection() {
  var selectAll = document.getElementById('selectAll');
  var bulkBar = document.getElementById('bulkBar');
  var bulkCount = document.getElementById('bulkCount');
  if (!selectAll || !bulkBar || !bulkCount) return;

  function updateBulkBar() {
    var checks = document.querySelectorAll('.file-check');
    var checked = document.querySelectorAll('.file-check:checked');
    var count = checked.length;

    if (count > 0) {
      bulkBar.classList.add('visible');
      bulkCount.textContent = count + '개 선택';
    } else {
      bulkBar.classList.remove('visible');
    }
    selectAll.checked = checks.length > 0 && checked.length === checks.length;
    selectAll.indeterminate = checked.length > 0 && checked.length < checks.length;
  }

  selectAll.addEventListener('change', function () {
    var checks = document.querySelectorAll('.file-check');
    checks.forEach(function (c) { c.checked = selectAll.checked; });
    updateBulkBar();
  });

  var tbody = document.querySelector('.file-table tbody');
  if (tbody) {
    tbody.addEventListener('change', function (e) {
      if (e.target.classList.contains('file-check')) updateBulkBar();
    });
  }
}

/* ============================================================
 * 6. Column drag reorder
 * ============================================================ */

function initColumnDragReorder() {
  var table = document.querySelector('.file-table');
  if (!table) return;

  var thead = table.querySelector('thead tr');
  var tbody = table.querySelector('tbody');
  if (!thead || !tbody) return;

  var dragSrcTh = null;

  thead.querySelectorAll('th[draggable="true"]').forEach(function (th) {
    th.addEventListener('dragstart', function (e) {
      if (e.target.classList.contains('col-resize-handle')) {
        e.preventDefault();
        return;
      }
      dragSrcTh = th;
      th.style.opacity = '0.4';
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', th.dataset.col);
    });

    th.addEventListener('dragend', function () {
      dragSrcTh = null;
      th.style.opacity = '';
      thead.querySelectorAll('th').forEach(function (h) {
        h.classList.remove('drag-over');
      });
    });

    th.addEventListener('dragover', function (e) {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      if (th !== dragSrcTh && th.hasAttribute('draggable')) {
        th.classList.add('drag-over');
      }
    });

    th.addEventListener('dragleave', function () {
      th.classList.remove('drag-over');
    });

    th.addEventListener('drop', function (e) {
      e.preventDefault();
      th.classList.remove('drag-over');
      if (!dragSrcTh || dragSrcTh === th) return;

      var allTh = Array.from(thead.children);
      var srcIdx = allTh.indexOf(dragSrcTh);
      var dstIdx = allTh.indexOf(th);
      if (srcIdx < 0 || dstIdx < 0) return;

      // Swap header cells
      if (srcIdx < dstIdx) {
        thead.insertBefore(dragSrcTh, th.nextSibling);
      } else {
        thead.insertBefore(dragSrcTh, th);
      }

      // Swap body cells in each row
      tbody.querySelectorAll('tr').forEach(function (row) {
        var cells = Array.from(row.children);
        var srcCell = cells[srcIdx];
        var dstCell = cells[dstIdx];
        if (srcIdx < dstIdx) {
          row.insertBefore(srcCell, dstCell.nextSibling);
        } else {
          row.insertBefore(srcCell, dstCell);
        }
      });
    });
  });
}

/* ============================================================
 * 7. Column resize
 * ============================================================ */

function initColumnResize() {
  var table = document.querySelector('.file-table');
  if (!table) return;

  var thead = table.querySelector('thead tr');
  if (!thead) return;

  thead.querySelectorAll('.col-resize-handle').forEach(function (handle) {
    handle.addEventListener('mousedown', function (e) {
      e.preventDefault();
      e.stopPropagation();

      var th = handle.parentElement;
      // Find next resizable sibling (skip action column)
      var nextTh = th.nextElementSibling;
      while (nextTh && !nextTh.hasAttribute('draggable')) {
        nextTh = nextTh.nextElementSibling;
      }
      if (!nextTh) return;

      th.removeAttribute('draggable');
      var startX = e.clientX;
      var startW = th.offsetWidth;
      var nextStartW = nextTh.offsetWidth;
      handle.classList.add('active');
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';

      function onMove(ev) {
        var diff = ev.clientX - startX;
        var newW = Math.max(40, startW + diff);
        var newNextW = Math.max(40, nextStartW - diff);
        th.style.width = newW + 'px';
        nextTh.style.width = newNextW + 'px';
      }

      function onUp() {
        handle.classList.remove('active');
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
        th.setAttribute('draggable', 'true');
        document.removeEventListener('mousemove', onMove);
        document.removeEventListener('mouseup', onUp);
      }

      document.addEventListener('mousemove', onMove);
      document.addEventListener('mouseup', onUp);
    });

    // Prevent drag when clicking resize handle
    handle.addEventListener('dragstart', function (e) { e.preventDefault(); });
  });
}

/* ============================================================
 * 8. Global click handler (close all dropdowns on outside click)
 *    -- 이미 initCloudDropdowns() 내부에서 document click 이벤트로 처리됨.
 *    -- 추가적인 글로벌 닫기가 필요한 경우 이 함수를 호출한다.
 * ============================================================ */

function initGlobalClickHandler() {
  // closeAllDropdowns 는 이미 전역 함수로 선언되어 있으므로,
  // 추가적인 커스텀 드롭다운이 있을 때 여기에 등록한다.
  document.addEventListener('click', function (e) {
    // 커스텀 드롭다운이 추가될 경우 여기서 닫기 처리
    if (
      !e.target.closest('.action-menu') &&
      !e.target.closest('.create-menu') &&
      !e.target.closest('.sort-menu') &&
      !e.target.closest('[data-action-toggle]')
    ) {
      closeAllDropdowns();
    }
  });
}

/* ============================================================
 * 9. Cloud sub-navigation
 * ============================================================ */

/**
 * switchCloudContent - Cloud 하위 뷰 전환 (내 파일, 공유, 동기화, 휴지통)
 * @param {string} viewName - 활성화할 cloud-view 이름 (e.g. 'my-files', 'shared', 'sync', 'trash')
 */
function switchCloudContent(viewName) {
  // 사이드바 cloud-sub-item 활성화 동기화
  document.querySelectorAll('.cloud-sub-item').forEach(function (item) {
    item.classList.toggle('active', item.dataset.cloudView === viewName);
  });

  // 모바일 Cloud 서브내비 동기화 (Web 뷰 전용, 없으면 무시)
  document.querySelectorAll('#cloudMobileSubnav button[data-cloud-view]').forEach(function (btn) {
    btn.classList.toggle('active', btn.dataset.cloudView === viewName);
  });
}

/**
 * initCloudSubnav - Cloud 서브 내비게이션 클릭 이벤트 바인딩
 */
function initCloudSubnav() {
  // 사이드바 cloud-sub-item 클릭
  document.querySelectorAll('.cloud-sub-item').forEach(function (item) {
    item.addEventListener('click', function () {
      switchCloudContent(item.dataset.cloudView);
    });
  });

  // 모바일 Cloud 서브내비 버튼 (Web 뷰 전용)
  document.querySelectorAll('#cloudMobileSubnav button[data-cloud-view]').forEach(function (btn) {
    btn.addEventListener('click', function () {
      switchCloudContent(btn.dataset.cloudView);
    });
  });
}

/* ============================================================
 * 편의 함수: 모든 공통 UI 초기화를 한 번에 호출
 * ============================================================ */

function initCommonUI() {
  initSidebar();
  initTabSwitching();
  initTheme();
  initCloudDropdowns();
  initBulkSelection();
  initColumnDragReorder();
  initColumnResize();
  initCloudSubnav();
}
