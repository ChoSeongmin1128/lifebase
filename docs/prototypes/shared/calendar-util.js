/**
 * calendar-util.js
 *
 * LifeBase 프로토타입 캘린더 공통 유틸리티.
 * lifebase-web-view.html, lifebase-desktop-app-view.html 에서
 * 동일하게 사용하는 날짜 계산, 색상, 필터, 레인 배정 함수를 모아둔다.
 *
 * 전역 변수/함수로 노출되며 모듈 시스템을 사용하지 않는다.
 * 이 파일을 <script src="shared/calendar-util.js"> 로 로드한 뒤,
 * 각 프로토타입 HTML 에서 YEAR, rangeEvents, singleEvents, todoItems,
 * activeAccount, showCompletedTodos 등 상태 변수를 선언해야 한다.
 */

/* ===== Color map ===== */

var colorHexMap = {
  teal: '#1b998b', blue: '#2563eb', red: '#dc2626',
  yellow: '#d97706', green: '#16a34a', lavender: '#6d6492'
};

function hexToRgba(hex, alpha) {
  var r = parseInt(hex.slice(1, 3), 16);
  var g = parseInt(hex.slice(3, 5), 16);
  var b = parseInt(hex.slice(5, 7), 16);
  return 'rgba(' + r + ',' + g + ',' + b + ',' + alpha + ')';
}

/* ===== Date helpers ===== */

/**
 * 날짜를 YYYY-MM-DD 문자열로 변환한다.
 * @param {number} y - 연도
 * @param {number} m - 월 (1-12)
 * @param {number} d - 일
 * @returns {string}
 */
function dateKey(y, m, d) {
  var mm = String(m).padStart(2, '0');
  var dd = String(d).padStart(2, '0');
  return y + '-' + mm + '-' + dd;
}

/**
 * YYYY-MM-DD 문자열을 Date 객체로 파싱한다.
 * @param {string} key
 * @returns {Date}
 */
function parseDate(key) {
  var parts = key.split('-').map(Number);
  return new Date(parts[0], parts[1] - 1, parts[2]);
}

/**
 * YYYY-MM-DD 문자열을 해당 연도 1월 1일 기준 일 인덱스로 변환한다.
 * YEAR 전역 변수에 의존한다.
 * @param {string} dateStr
 * @returns {number}
 */
function toIndex(dateStr) {
  var date = parseDate(dateStr);
  var start = new Date(YEAR, 0, 1);
  return Math.floor((date - start) / 86400000) + 1;
}

/**
 * 주어진 날짜가 속한 주의 월요일을 반환한다 (ISO 주 기준).
 * @param {Date} date
 * @returns {Date}
 */
function startOfWeek(date) {
  var d = new Date(date);
  var day = d.getDay();
  var diff = day === 0 ? -6 : 1 - day;
  d.setDate(d.getDate() + diff);
  d.setHours(0, 0, 0, 0);
  return d;
}

/**
 * 날짜에 n일을 더한 새 Date를 반환한다.
 * @param {Date} date
 * @param {number} days
 * @returns {Date}
 */
function addDays(date, days) {
  var d = new Date(date);
  d.setDate(d.getDate() + days);
  return d;
}

/**
 * 두 Date가 같은 날인지 비교한다.
 * @param {Date} a
 * @param {Date} b
 * @returns {boolean}
 */
function isSameDay(a, b) {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

/**
 * 월 인덱스(0-11)를 한국어 레이블로 변환한다.
 * @param {number} monthIndex
 * @returns {string}
 */
function monthLabel(monthIndex) {
  return (monthIndex + 1) + '월';
}

/* ===== Account filter ===== */

/**
 * 현재 선택된 계정(activeAccount)에 해당하는 항목인지 판별한다.
 * activeAccount 전역 변수에 의존한다.
 * @param {{ account: string }} item
 * @returns {boolean}
 */
function passAccount(item) {
  return activeAccount === 'all' || item.account === activeAccount;
}

/**
 * rangeEvents 중 현재 계정 필터를 통과하는 항목만 반환한다.
 * rangeEvents 전역 변수에 의존한다.
 * @returns {Array}
 */
function filteredRangeEvents() {
  return rangeEvents.filter(passAccount);
}

/**
 * singleEvents 중 특정 날짜 + 현재 계정 필터를 통과하는 항목만 반환한다.
 * singleEvents 전역 변수에 의존한다.
 * @param {string} key - YYYY-MM-DD
 * @returns {Array}
 */
function filteredSinglesForDate(key) {
  return singleEvents.filter(function (event) {
    return event.date === key && passAccount(event);
  });
}

/**
 * todoItems 중 특정 날짜 + 계정 필터 + 완료 표시 여부를 통과하는 항목만 반환한다.
 * todoItems, showCompletedTodos 전역 변수에 의존한다.
 * @param {string} key - YYYY-MM-DD
 * @returns {Array}
 */
function todosForDate(key) {
  return todoItems.filter(function (todo) {
    return todo.date === key && passAccount(todo) && (showCompletedTodos || !todo.done);
  });
}

/* ===== Lane assignment (timeline) ===== */

/**
 * 기간 이벤트 목록에 대해 겹치지 않는 레인(행)을 배정한다.
 * @param {Array<{ start: string, end: string }>} events
 * @returns {Map} event -> lane(number)
 */
function assignLanes(events) {
  var sorted = events.slice().sort(function (a, b) {
    var startDiff = toIndex(a.start) - toIndex(b.start);
    if (startDiff !== 0) return startDiff;
    return toIndex(b.end) - toIndex(a.end);
  });

  var lanes = new Map();
  var active = [];
  sorted.forEach(function (event) {
    var start = toIndex(event.start);
    var end = toIndex(event.end);

    for (var i = active.length - 1; i >= 0; i -= 1) {
      if (active[i].end < start) active.splice(i, 1);
    }

    var used = new Set(active.map(function (entry) { return entry.lane; }));
    var lane = 0;
    while (used.has(lane)) lane += 1;
    lanes.set(event, lane);
    active.push({ lane: lane, end: end });
  });

  return lanes;
}

/**
 * 특정 날짜에 걸쳐 있는 기간 이벤트를 레인 순으로 반환한다.
 * @param {string} key - YYYY-MM-DD
 * @param {Array<{ start: string, end: string }>} events
 * @param {Map} laneMap - assignLanes()의 반환값
 * @returns {Array}
 */
function rangesForDate(key, events, laneMap) {
  var idx = toIndex(key);
  return events
    .filter(function (event) {
      return idx >= toIndex(event.start) && idx <= toIndex(event.end);
    })
    .sort(function (a, b) {
      return (laneMap.get(a) || 0) - (laneMap.get(b) || 0);
    });
}
