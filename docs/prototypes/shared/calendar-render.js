/**
 * LifeBase Prototype — Shared Calendar Rendering
 *
 * 프로토타입 HTML 파일(web, desktop)에서 공통으로 사용하는
 * 캘린더 렌더링 함수를 한 곳에서 관리한다.
 *
 * 의존성 (이 파일보다 먼저 로드해야 함):
 *   - shared/demo-data.js    (YEAR, DOW_KR, demoToday, accountLabel, holidays,
 *                             rangeEvents, singleEvents, timedEvents, todoItems)
 *   - shared/calendar-util.js (dateKey, parseDate, toIndex, startOfWeek, addDays,
 *                             isSameDay, monthLabel, colorHexMap, hexToRgba,
 *                             passAccount, filteredRangeEvents, filteredSinglesForDate,
 *                             todosForDate, assignLanes, rangesForDate)
 *
 * <script src="shared/calendar-render.js"> 로 로드하면 전역 함수로 노출된다.
 */

/* ================================================================
 * Calendar state variables
 * ================================================================ */

var activeAccount = 'all';
var showCompletedTodos = false;
var activeCalendarView = 'year';
var activeYearMode = 'compact';
var weekHourStart = 8;
var weekHourEnd = 22;
var currentMonth = demoToday.getMonth();
var weekStart = startOfWeek(demoToday);

/* ================================================================
 * Year Compact view
 * ================================================================ */

function renderYearCompact(events, laneMap) {
  const container = document.getElementById('yearGrid');
  container.innerHTML = '';

  for (let month = 1; month <= 12; month += 1) {
    const card = document.createElement('div');
    card.className = 'month-card';

    const head = document.createElement('div');
    head.className = 'month-head';
    head.textContent = `${month}월`;

    const week = document.createElement('div');
    week.className = 'mini-week';
    ['일', '월', '화', '수', '목', '금', '토'].forEach((w) => {
      const s = document.createElement('span');
      s.textContent = w;
      week.appendChild(s);
    });

    const days = document.createElement('div');
    days.className = 'mini-days';

    const firstDow = new Date(YEAR, month - 1, 1).getDay();
    const maxDay = new Date(YEAR, month, 0).getDate();

    for (let i = 0; i < firstDow; i += 1) {
      const empty = document.createElement('div');
      empty.className = 'mini-day empty';
      days.appendChild(empty);
    }

    for (let day = 1; day <= maxDay; day += 1) {
      const key = dateKey(YEAR, month, day);
      const cell = document.createElement('div');
      cell.className = 'mini-day';
      if (isSameDay(parseDate(key), demoToday)) cell.classList.add('secondary-focus');

      const num = document.createElement('span');
      num.className = 'mini-num';
      num.textContent = day;
      cell.appendChild(num);

      const ranges = rangesForDate(key, events, laneMap);
      const visibleRanges = ranges.filter((event) => (laneMap.get(event) || 0) < 3);
      const hiddenRanges = ranges.length - visibleRanges.length;
      const singles = filteredSinglesForDate(key);
      const todos = todosForDate(key);

      let shownSingles = 0;
      let shownTodos = 0;
      const markerColors = visibleRanges.map((range) => range.color);
      if (markerColors.length < 3 && singles.length > 0) {
        markerColors.push(singles[0].color);
        shownSingles = 1;
      }
      if (markerColors.length < 3 && todos.length > 0) {
        markerColors.push('todo');
        shownTodos = 1;
      }

      if (markerColors.length > 0) {
        const bars = document.createElement('div');
        bars.className = 'mini-bars';
        markerColors.forEach((color) => {
          const bar = document.createElement('span');
          bar.className = `mini-bar bar-${color}`;
          bars.appendChild(bar);
        });
        cell.appendChild(bars);
      }

      const hiddenEvents = hiddenRanges + Math.max(0, singles.length - shownSingles);
      const hiddenTodos = Math.max(0, todos.length - shownTodos);
      const hiddenTotal = hiddenEvents + hiddenTodos;
      if (hiddenTotal > 0) {
        const chips = document.createElement('div');
        chips.className = 'mini-chips';
        const chip = document.createElement('span');
        chip.className = 'tiny-chip chip-more';
        chip.textContent = `+${hiddenTotal}`;
        chips.appendChild(chip);
        cell.appendChild(chips);
      }

      days.appendChild(cell);
    }

    const total = firstDow + maxDay;
    const extra = (7 - (total % 7)) % 7;
    for (let i = 0; i < extra; i += 1) {
      const empty = document.createElement('div');
      empty.className = 'mini-day empty';
      days.appendChild(empty);
    }

    card.appendChild(head);
    card.appendChild(week);
    card.appendChild(days);
    container.appendChild(card);
  }
}

/* ================================================================
 * Year Timeline view
 * ================================================================ */

function getRangePosition(range, key, month, day) {
  if (range.start === key && range.end === key) return 'single';
  if (range.start === key) return 'start';
  if (range.end === key) return 'end';
  const rangeStart = parseDate(range.start);
  if (day === 1 && rangeStart.getMonth() + 1 < month) return 'start';
  return 'mid';
}

function buildYearTimelineCellHTML(month, day, events, laneMap) {
  const maxDays = new Date(YEAR, month, 0).getDate();
  if (day > maxDays) return '<div class="yt-cell empty"></div>';

  const key = dateKey(YEAR, month, day);
  const dow = new Date(YEAR, month - 1, day).getDay();
  const weekend = dow === 0 || dow === 6;
  const isToday = isSameDay(parseDate(key), demoToday);
  const holiday = holidays[key];
  const singles = filteredSinglesForDate(key);
  const todos = todosForDate(key);

  const allRanges = rangesForDate(key, events, laneMap);
  const visibleRanges = allRanges.filter((event) => (laneMap.get(event) || 0) < 3);
  const hiddenRanges = allRanges.filter((event) => (laneMap.get(event) || 0) >= 3);

  let cls = 'yt-cell';
  if (weekend) cls += ' weekend';
  if (isToday) cls += ' today';

  let html = `<div class="${cls}" title="${month}월 ${day}일 (${DOW_KR[dow]})">`;

  visibleRanges.forEach((range) => {
    const position = getRangePosition(range, key, month, day);
    const lane = laneMap.get(range) || 0;
    let barClass = `yt-range-bar bar-${range.color} lane-${lane}`;
    if (position === 'start') barClass += ' start';
    else if (position === 'end') barClass += ' end';
    else if (position === 'single') barClass += ' single';
    html += `<div class="${barClass}"></div>`;
  });

  html += `<span class="yt-label${dow === 0 ? ' sun' : ''}${dow === 6 ? ' sat' : ''}">${day}</span>`;

  let hiddenSingles = [];
  let hiddenTodos = [];
  if (holiday) {
    html += `<span class="yt-event-text color-holiday">${holiday}</span>`;
    hiddenSingles = singles;
    hiddenTodos = todos;
  } else if (visibleRanges.length > 0) {
    const startRange = visibleRanges.find((range) => {
      const position = getRangePosition(range, key, month, day);
      return position === 'start' || position === 'single';
    });

    if (startRange) {
      html += `<span class="yt-event-text color-${startRange.color}">${startRange.title}</span>`;
      hiddenSingles = singles;
      hiddenTodos = todos;
    } else if (singles.length > 0) {
      html += `<span class="yt-event-text color-${singles[0].color}">${singles[0].title}</span>`;
      hiddenSingles = singles.slice(1);
      hiddenTodos = todos;
    } else if (todos.length > 0) {
      html += `<span class="yt-todo-pill${todos[0].done ? ' done' : ''}">${todos[0].title}</span>`;
      hiddenTodos = todos.slice(1);
    } else if (day === 1) {
      html += `<span class="yt-event-text color-${visibleRanges[0].color} continued">← ${visibleRanges[0].title}</span>`;
    }
  } else if (singles.length > 0) {
    html += `<span class="yt-event-text color-${singles[0].color}">${singles[0].title}</span>`;
    hiddenSingles = singles.slice(1);
    hiddenTodos = todos;
  } else if (todos.length > 0) {
    html += `<span class="yt-todo-pill${todos[0].done ? ' done' : ''}">${todos[0].title}</span>`;
    hiddenTodos = todos.slice(1);
  }

  const hiddenTotal = hiddenRanges.length + hiddenSingles.length + hiddenTodos.length;
  if (hiddenTotal > 0) html += `<span class="yt-more-badge">+${hiddenTotal}</span>`;

  html += '</div>';
  return html;
}

function renderYearTimeline(events, laneMap) {
  const head = document.getElementById('yearTimelineHead');
  const body = document.getElementById('yearTimelineBody');

  head.innerHTML = '';
  for (let month = 1; month <= 12; month += 1) {
    head.innerHTML += `<th>${month}월</th>`;
  }

  body.innerHTML = '';
  for (let day = 1; day <= 31; day += 1) {
    const row = document.createElement('tr');
    for (let month = 1; month <= 12; month += 1) {
      const td = document.createElement('td');
      td.className = 'month-c';
      td.innerHTML = buildYearTimelineCellHTML(month, day, events, laneMap);
      row.appendChild(td);
    }
    body.appendChild(row);
  }
}

function syncYearTimelineRowHeight() {
  const wrapper = document.querySelector('#year-mode-timeline .year-timeline-wrap');
  const table = document.getElementById('yearTimelineTable');
  if (!wrapper || !table) return;

  // 타임라인이 보이지 않을 때는 높이 계산을 건너뛴다
  if (wrapper.clientHeight === 0) return;

  const isMobile = window.innerWidth <= 760;
  const headerCell = table.querySelector('thead th');
  const headerHeight = headerCell ? headerCell.getBoundingClientRect().height : 36;
  let available;
  if (isMobile) {
    const vh = window.innerHeight;
    available = vh - 64 - 50 - 60 - 16 - headerHeight;
  } else {
    available = wrapper.clientHeight - headerHeight - 2;
  }
  const rowHeight = available / 31;
  const minRow = isMobile ? 8 : 12;
  const clamped = Math.max(minRow, rowHeight);
  document.documentElement.style.setProperty('--year-row-height', `${clamped.toFixed(2)}px`);
}

/* ================================================================
 * Month view
 * ================================================================ */

function renderMonthView(events, laneMap) {
  const monthDays = document.getElementById('monthDays');
  monthDays.innerHTML = '';

  const firstDow = new Date(YEAR, currentMonth, 1).getDay();
  const maxDay = new Date(YEAR, currentMonth + 1, 0).getDate();
  const prevMonthMax = new Date(YEAR, currentMonth, 0).getDate();

  // Build flat array of all 42 day-cells with metadata
  const cells = [];
  for (let i = firstDow - 1; i >= 0; i -= 1) {
    const d = prevMonthMax - i;
    const dt = new Date(YEAR, currentMonth - 1, d);
    cells.push({ day: d, key: dateKey(dt.getFullYear(), dt.getMonth() + 1, dt.getDate()), outside: true });
  }
  for (let day = 1; day <= maxDay; day += 1) {
    cells.push({ day, key: dateKey(YEAR, currentMonth + 1, day), outside: false });
  }
  while (cells.length < 42) {
    const d = cells.length - (firstDow + maxDay) + 1;
    const dt = new Date(YEAR, currentMonth + 1, d);
    cells.push({ day: d, key: dateKey(dt.getFullYear(), dt.getMonth() + 1, dt.getDate()), outside: true });
  }

  // Render week-by-week
  for (let w = 0; w < 6; w++) {
    const weekCells = cells.slice(w * 7, w * 7 + 7);
    const weekRow = document.createElement('div');
    weekRow.className = 'month-week-row';

    // Find range events spanning this week
    const weekStartKey = weekCells[0].key;
    const weekEndKey = weekCells[6].key;
    const weekRanges = events.filter(ev => ev.start <= weekEndKey && ev.end >= weekStartKey);

    // Assign lanes for this week's range events
    const weekLanes = assignLanes(weekRanges);
    const maxLane = weekRanges.length > 0 ? Math.max(...Array.from(weekLanes.values())) : -1;
    const visibleLanes = Math.min(maxLane + 1, 3); // max 3 range bar rows
    const barRowHeight = 20;
    const rangePadding = visibleLanes > 0 ? visibleLanes * barRowHeight + 2 : 0;

    // Range event bars layer
    if (visibleLanes > 0) {
      const layer = document.createElement('div');
      layer.className = 'month-range-layer';
      layer.style.top = `24px`;

      weekRanges.forEach(ev => {
        const lane = weekLanes.get(ev) || 0;
        if (lane >= 3) return; // skip overflow lanes

        const evStart = ev.start < weekStartKey ? 0 : weekCells.findIndex(c => c.key === ev.start);
        const evEnd = ev.end > weekEndKey ? 6 : weekCells.findIndex(c => c.key === ev.end);
        if (evStart < 0 || evEnd < 0) return;

        const colStart = evStart + 1;
        const colEnd = evEnd + 2;
        const startsHere = ev.start >= weekStartKey;
        const endsHere = ev.end <= weekEndKey;

        const bar = document.createElement('div');
        const hex = colorHexMap[ev.color] || '#888';
        bar.className = 'month-range-bar';
        bar.style.gridColumn = `${colStart} / ${colEnd}`;
        bar.style.gridRow = `${lane + 1}`;
        bar.style.background = hexToRgba(hex, 0.18);
        bar.style.color = hex;
        bar.textContent = ev.title;

        if (startsHere && endsHere) bar.classList.add('single');
        else if (startsHere) bar.classList.add('start');
        else if (endsHere) bar.classList.add('end');
        else bar.classList.add('mid');

        layer.appendChild(bar);
      });
      weekRow.appendChild(layer);
    }

    // Day cells
    weekCells.forEach(c => {
      const cell = document.createElement('div');
      cell.className = 'day-cell';
      if (c.outside) cell.style.opacity = '0.35';
      if (!c.outside && isSameDay(parseDate(c.key), demoToday)) cell.classList.add('secondary-focus');

      const singles = c.outside ? [] : filteredSinglesForDate(c.key);
      const todos = c.outside ? [] : todosForDate(c.key);

      const items = [];
      singles.forEach(s => items.push({ type: 'event', color: s.color, title: s.title }));
      todos.forEach(t => items.push({ type: 'todo', title: t.title }));

      let html = `<div class="day-num">${c.day}</div>`;
      if (rangePadding > 0) html += `<div style="height:${rangePadding}px"></div>`;

      items.slice(0, 2).forEach(item => {
        if (item.type === 'todo') html += `<div class="todo-pill">${item.title}</div>`;
        else html += `<div class="event-pill ${item.color}">${item.title}</div>`;
      });

      const hidden = Math.max(0, items.length - 2);
      if (hidden > 0) html += `<div class="badge">+${hidden}</div>`;

      cell.innerHTML = html;
      weekRow.appendChild(cell);
    });

    monthDays.appendChild(weekRow);
  }
}

/* ================================================================
 * Week view
 * ================================================================ */

function renderWeekView() {
  const HOUR_START = weekHourStart;
  const HOUR_END = weekHourEnd;
  const HOUR_HEIGHT = 48;
  const dayNames = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

  const weekDates = [];
  for (let i = 0; i < 7; i++) weekDates.push(addDays(weekStart, i));

  // Header
  const header = document.getElementById('weekHeader');
  header.innerHTML = '<div></div>';
  weekDates.forEach((date, i) => {
    const dow = date.getDay();
    const isToday = isSameDay(date, demoToday);
    let cls = 'day-head';
    if (isToday) cls += ' today';
    if (dow === 0) cls += ' sun';
    if (dow === 6) cls += ' sat';
    header.innerHTML += `<div class="${cls}"><span class="day-name">${dayNames[i]}</span><span class="day-num">${date.getDate()}</span></div>`;
  });

  // All-day events (spanning bars + todos in grid layout)
  const allday = document.getElementById('weekAllday');
  allday.innerHTML = '';

  const weekKeys = weekDates.map(d => dateKey(d.getFullYear(), d.getMonth() + 1, d.getDate()));
  const weekStartKey = weekKeys[0];
  const weekEndKey = weekKeys[6];

  // Collect range events for this week
  const weekRangeEvts = rangeEvents.filter(ev => passAccount(ev) && ev.start <= weekEndKey && ev.end >= weekStartKey);
  const weekRangeLanes = assignLanes(weekRangeEvts);
  const maxRangeLane = weekRangeEvts.length > 0 ? Math.max(...Array.from(weekRangeLanes.values())) : -1;
  const visibleRangeLanes = Math.min(maxRangeLane + 1, 4);
  const barRowH = 20;

  // Range bar rows (each row = 1 grid row spanning label + 7 cols)
  for (let lane = 0; lane < visibleRangeLanes; lane++) {
    // Empty label cell for this lane row
    const laneLabel = document.createElement('div');
    laneLabel.className = 'wk-allday-lane-label';
    if (lane === 0) laneLabel.textContent = '종일';
    allday.appendChild(laneLabel);

    // 7 day cells for this lane row (as positioning context)
    for (let d = 0; d < 7; d++) {
      const cell = document.createElement('div');
      cell.className = 'wk-allday-bar-cell';
      allday.appendChild(cell);
    }
  }

  // Place range bars using grid-column + grid-row
  weekRangeEvts.forEach(ev => {
    const lane = weekRangeLanes.get(ev) || 0;
    if (lane >= visibleRangeLanes) return;

    const evStartIdx = ev.start < weekStartKey ? 0 : weekKeys.indexOf(ev.start);
    const evEndIdx = ev.end > weekEndKey ? 6 : weekKeys.indexOf(ev.end);
    if (evStartIdx < 0 || evEndIdx < 0) return;

    const startsHere = ev.start >= weekStartKey;
    const endsHere = ev.end <= weekEndKey;

    const bar = document.createElement('div');
    const hex = colorHexMap[ev.color] || '#888';
    bar.className = 'wk-range-bar';
    bar.style.background = hexToRgba(hex, 0.18);
    bar.style.color = hex;
    // grid-column: col 1 is label, cols 2-8 are days
    bar.style.gridColumn = `${evStartIdx + 2} / ${evEndIdx + 3}`;
    bar.style.gridRow = `${lane + 1}`;
    bar.textContent = ev.title;

    if (startsHere && endsHere) bar.classList.add('single');
    else if (startsHere) bar.classList.add('start');
    else if (endsHere) bar.classList.add('end');
    else bar.classList.add('mid');

    allday.appendChild(bar);
  });

  // Todo row (label + 7 cells)
  const todoLabel = document.createElement('div');
  todoLabel.className = 'wk-allday-lane-label';
  if (visibleRangeLanes === 0) todoLabel.textContent = '종일';
  allday.appendChild(todoLabel);

  weekDates.forEach(date => {
    const key = dateKey(date.getFullYear(), date.getMonth() + 1, date.getDate());
    const cell = document.createElement('div');
    cell.className = 'week-allday-cell';
    todoItems
      .filter(todo => todo.date === key && passAccount(todo) && (showCompletedTodos || !todo.done))
      .forEach(todo => {
        cell.innerHTML += `<span class="wk-todo-chip">${todo.title}</span>`;
      });
    allday.appendChild(cell);
  });

  // Time grid (with 30min top padding for scroll context)
  const grid = document.getElementById('weekGrid');
  grid.innerHTML = '';
  const gridStart = HOUR_START;
  // Add 30min spacer row at top (no label, lighter border)
  if (HOUR_START > 0) {
    const spacerLabel = document.createElement('div');
    spacerLabel.className = 'week-time-label week-time-spacer';
    grid.appendChild(spacerLabel);
    for (let d = 0; d < 7; d++) {
      const spacerCell = document.createElement('div');
      spacerCell.className = 'week-time-cell week-time-spacer';
      const dow = weekDates[d].getDay();
      if (dow === 0 || dow === 6) spacerCell.classList.add('weekend');
      grid.appendChild(spacerCell);
    }
  }
  for (let h = gridStart; h < HOUR_END; h++) {
    const label = document.createElement('div');
    label.className = 'week-time-label';
    label.innerHTML = `<span>${String(h).padStart(2, '0')}</span>`;
    grid.appendChild(label);

    for (let d = 0; d < 7; d++) {
      const cell = document.createElement('div');
      cell.className = 'week-time-cell';
      const dow = weekDates[d].getDay();
      if (dow === 0 || dow === 6) cell.classList.add('weekend');
      cell.dataset.day = d;
      cell.dataset.hour = h;
      grid.appendChild(cell);
    }
  }

  // Place timed events as positioned blocks
  weekDates.forEach((date, dayIdx) => {
    const key = dateKey(date.getFullYear(), date.getMonth() + 1, date.getDate());
    const dayEvents = timedEvents.filter((e) => passAccount(e) && e.date === key);
    if (dayEvents.length === 0) return;

    // Parse events with start/end minutes for overlap detection
    const parsed = dayEvents.map((event) => {
      const [hh, mm] = event.time.split(':').map(Number);
      const startMin = hh * 60 + mm;
      const duration = event.duration || 1;
      const endMin = startMin + duration * 60;
      return { event, hh, mm, startMin, endMin, duration };
    }).filter((p) => {
      const startHour = p.hh + p.mm / 60;
      return startHour >= HOUR_START && startHour < HOUR_END;
    });

    // Assign columns for overlapping events
    parsed.sort((a, b) => a.startMin - b.startMin || a.endMin - b.endMin);
    const columns = [];
    parsed.forEach((p) => {
      let col = 0;
      while (columns[col] && columns[col] > p.startMin) col++;
      p.col = col;
      columns[col] = p.endMin;
    });
    // Calculate total columns per overlap group
    parsed.forEach((p) => {
      const overlapping = parsed.filter((q) => q.startMin < p.endMin && q.endMin > p.startMin);
      p.totalCols = Math.max(...overlapping.map((q) => q.col + 1));
    });

    parsed.forEach((p) => {
      const top = (p.mm / 60) * HOUR_HEIGHT;
      const height = Math.max(p.duration * HOUR_HEIGHT - 2, 18);

      const block = document.createElement('div');
      block.className = `wk-event-block ${p.event.color}`;
      block.style.top = `${top}px`;
      block.style.height = `${height}px`;
      if (p.totalCols > 1) {
        const widthPct = 100 / p.totalCols;
        block.style.left = `${widthPct * p.col}%`;
        block.style.width = `${widthPct}%`;
        block.style.right = 'auto';
      }
      const endHH = String(Math.floor(p.endMin / 60)).padStart(2, '0');
      const endMM = String(p.endMin % 60).padStart(2, '0');
      block.innerHTML = `<div class="wk-time">${p.event.time}-${endHH}:${endMM}</div><div class="wk-title">${p.event.title}</div>`;

      const targetCell = grid.querySelector(`.week-time-cell[data-day="${dayIdx}"][data-hour="${p.hh}"]`);
      if (targetCell) targetCell.appendChild(block);
    });
  });

  // Scroll to top (30min spacer is already visible)
  const scrollEl = document.getElementById('weekGridScroll');
  scrollEl.scrollTop = 0;
}

/* ================================================================
 * Agenda view
 * ================================================================ */

let agendaAnchor = new Date(demoToday);
let agendaRangeBefore = 14;
let agendaRangeAfter = 14;

function buildAgendaDay(date) {
  const key = dateKey(date.getFullYear(), date.getMonth() + 1, date.getDate());
  const isToday = isSameDay(date, demoToday);
  const dowNames = ['일', '월', '화', '수', '목', '금', '토'];

  const dayEl = document.createElement('div');
  dayEl.className = 'agenda-day';

  const header = document.createElement('div');
  header.className = 'agenda-day-header' + (isToday ? ' today' : '');
  header.textContent = `${date.getMonth() + 1}/${date.getDate()} (${dowNames[date.getDay()]})`;
  dayEl.appendChild(header);

  let hasContent = false;

  // All-day range events
  rangeEvents.filter(ev => passAccount(ev) && key >= ev.start && key <= ev.end).forEach(ev => {
    hasContent = true;
    const hex = colorHexMap[ev.color] || '#888';
    const el = document.createElement('div');
    el.className = 'agenda-allday';
    el.style.background = hexToRgba(hex, 0.15);
    el.style.color = hex;
    el.style.borderLeftColor = hex;
    el.textContent = ev.title;
    dayEl.appendChild(el);
  });

  // All-day single events (no time field in singleEvents)
  singleEvents.filter(ev => ev.date === key && passAccount(ev)).forEach(ev => {
    hasContent = true;
    const hex = colorHexMap[ev.color] || '#888';
    const el = document.createElement('div');
    el.className = 'agenda-allday';
    el.style.background = hexToRgba(hex, 0.15);
    el.style.color = hex;
    el.style.borderLeftColor = hex;
    el.textContent = ev.title;
    dayEl.appendChild(el);
  });

  // Timed events
  const dayTimed = timedEvents.filter(ev => ev.date === key && passAccount(ev));
  dayTimed.sort((a, b) => a.time.localeCompare(b.time));
  dayTimed.forEach(ev => {
    hasContent = true;
    const [hh, mm] = ev.time.split(':').map(Number);
    const endMin = hh * 60 + mm + ev.duration * 60;
    const endH = String(Math.floor(endMin / 60)).padStart(2, '0');
    const endM = String(endMin % 60).padStart(2, '0');
    const hex = colorHexMap[ev.color] || '#888';
    const el = document.createElement('div');
    el.className = 'agenda-event';
    el.innerHTML = `<span class="agenda-dot" style="background:${hex}"></span><span class="agenda-time">${ev.time} - ${endH}:${endM}</span><span>${ev.title}</span>`;
    dayEl.appendChild(el);
  });

  // Todos
  todoItems.filter(t => t.date === key && passAccount(t) && (showCompletedTodos || !t.done)).forEach(t => {
    hasContent = true;
    const el = document.createElement('div');
    el.className = 'agenda-todo';
    el.textContent = t.title;
    dayEl.appendChild(el);
  });

  if (!hasContent) return null;

  return dayEl;
}

function renderAgendaView() {
  const wrap = document.getElementById('agendaWrap');
  wrap.innerHTML = '';
  const startDate = addDays(agendaAnchor, -agendaRangeBefore);
  const totalDays = agendaRangeBefore + agendaRangeAfter + 1;

  let hasAny = false;
  for (let i = 0; i < totalDays; i++) {
    const el = buildAgendaDay(addDays(startDate, i));
    if (el) { wrap.appendChild(el); hasAny = true; }
  }

  if (!hasAny) {
    const empty = document.createElement('div');
    empty.className = 'agenda-empty-state';
    empty.textContent = '이 기간에 일정이 없습니다';
    wrap.appendChild(empty);
  }

  // scroll to today
  const todayEl = wrap.querySelector('.agenda-day-header.today');
  if (todayEl) todayEl.scrollIntoView({ block: 'start' });

  // infinite scroll
  wrap.onscroll = () => {
    if (wrap.scrollTop < 80) {
      agendaRangeBefore += 7;
      const oldH = wrap.scrollHeight;
      const frag = document.createDocumentFragment();
      const newStart = addDays(agendaAnchor, -agendaRangeBefore);
      for (let i = 0; i < 7; i++) {
        const el = buildAgendaDay(addDays(newStart, i));
        if (el) frag.appendChild(el);
      }
      if (frag.childNodes.length > 0) {
        wrap.prepend(frag);
        wrap.scrollTop += wrap.scrollHeight - oldH;
      }
    }
    if (wrap.scrollTop + wrap.clientHeight > wrap.scrollHeight - 80) {
      agendaRangeAfter += 7;
      const end = addDays(agendaAnchor, agendaRangeAfter - 6);
      for (let i = 0; i < 7; i++) {
        const el = buildAgendaDay(addDays(end, i));
        if (el) wrap.appendChild(el);
      }
    }
  };
}

/* ================================================================
 * Todo panel
 * ================================================================ */

function renderTodoPanel() {
  const container = document.getElementById('todoListItems');
  const todos = [...todoItems]
    .filter((todo) => passAccount(todo) && (showCompletedTodos || !todo.done))
    .sort((a, b) => a.date.localeCompare(b.date));

  container.innerHTML = '';
  if (todos.length === 0) {
    container.innerHTML = '<div class="task-meta">표시할 Todo가 없습니다.</div>';
    return;
  }

  todos.forEach((todo) => {
    const item = document.createElement('div');
    item.className = `task-item${todo.done ? ' soft-secondary' : ''}`;
    item.innerHTML = `
      <div class="task-row">
        <span class="check${todo.done ? ' done' : ''}"></span>
        <div>
          <div class="task-title${todo.done ? ' done' : ''}">${todo.title}</div>
          <div class="task-meta">Due ${todo.date} · ${accountLabel[todo.account]}</div>
        </div>
      </div>
    `;
    container.appendChild(item);
  });
}

/* ================================================================
 * Calendar dispatch & navigation
 * ================================================================ */

function renderCalendar() {
  const events = filteredRangeEvents();
  const laneMap = assignLanes(events);
  renderYearCompact(events, laneMap);
  renderYearTimeline(events, laneMap);
  renderMonthView(events, laneMap);
  renderWeekView();
  renderAgendaView();
  renderTodoPanel();
  updateCalendarRangeLabel();
  requestAnimationFrame(syncYearTimelineRowHeight);
}

function updateCalendarRangeLabel() {
  const label = document.getElementById('calendarRangeLabel');
  if (activeCalendarView === 'month') {
    label.textContent = `${YEAR}년 ${monthLabel(currentMonth)}`;
    return;
  }
  if (activeCalendarView === 'week') {
    const end = addDays(weekStart, 6);
    label.textContent = `${weekStart.getMonth() + 1}/${weekStart.getDate()} - ${end.getMonth() + 1}/${end.getDate()}`;
    return;
  }
  if (activeCalendarView === 'agenda') {
    label.textContent = 'Agenda';
    return;
  }
  label.textContent = `${YEAR}년`;
}

function syncTodoToggleText() {
  const toggle = document.getElementById('todoDoneToggle');
  toggle.textContent = showCompletedTodos ? '완료 Todo 표시 중' : '완료 Todo 숨김';
}

function syncTimelineFocusClass() {
  const calendarPanel = document.getElementById('tab-calendar');
  calendarPanel.classList.toggle('timeline-focus', activeYearMode === 'timeline');
}

var calViewIcons = {
  'year-compact': '<svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="1" y="1" width="6" height="6" rx="1"/><rect x="9" y="1" width="6" height="6" rx="1"/><rect x="1" y="9" width="6" height="6" rx="1"/><rect x="9" y="9" width="6" height="6" rx="1"/></svg>',
  'year-timeline': '<svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="1" y1="3" x2="15" y2="3"/><line x1="1" y1="8" x2="11" y2="8"/><line x1="1" y1="13" x2="13" y2="13"/></svg>',
  'month': '<svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="1" y="1" width="14" height="14" rx="2"/><line x1="1" y1="5.5" x2="15" y2="5.5"/><line x1="5.5" y1="5.5" x2="5.5" y2="15"/><line x1="10.5" y1="5.5" x2="10.5" y2="15"/><line x1="1" y1="10" x2="15" y2="10"/></svg>',
  'week': '<svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="1" y="1" width="14" height="14" rx="2"/><line x1="1" y1="5" x2="15" y2="5"/><line x1="5" y1="5" x2="5" y2="15"/><line x1="9" y1="5" x2="9" y2="15"/><line x1="13" y1="5" x2="13" y2="15"/></svg>',
  'agenda': '<svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="5" y1="3" x2="14" y2="3"/><line x1="5" y1="8" x2="14" y2="8"/><line x1="5" y1="13" x2="14" y2="13"/><circle cx="2" cy="3" r="1" fill="currentColor" stroke="none"/><circle cx="2" cy="8" r="1" fill="currentColor" stroke="none"/><circle cx="2" cy="13" r="1" fill="currentColor" stroke="none"/></svg>'
};

function updateCalViewLabel() {
  const calViewBtn = document.getElementById('calViewBtn');
  const viewNames = { year: '연간', month: '월간', week: '주간', agenda: '일정' };
  let label = viewNames[activeCalendarView];
  let iconKey = activeCalendarView;
  if (activeCalendarView === 'year') {
    label = activeYearMode === 'timeline' ? '연간 · 타임라인' : '연간 · 달력';
    iconKey = 'year-' + activeYearMode;
  }
  calViewBtn.querySelector('.cal-view-btn-icon').innerHTML = calViewIcons[iconKey] || '';
  calViewBtn.querySelector('.cal-view-btn-label').textContent = label;
}

function switchCalendarView(cview, ymode) {
  activeCalendarView = cview;
  if (ymode) activeYearMode = ymode;

  const cviews = document.querySelectorAll('.calendar-view');
  cviews.forEach((v) => v.classList.remove('active'));
  document.getElementById(`view-${cview}`).classList.add('active');

  if (cview === 'year') {
    const yearModes = document.querySelectorAll('.year-mode');
    yearModes.forEach((m) => m.classList.remove('active'));
    document.getElementById(`year-mode-${activeYearMode}`).classList.add('active');
  }

  const calViewMenu = document.getElementById('calViewMenu');
  const calViewItems = calViewMenu.querySelectorAll('.cal-view-menu-item');
  calViewItems.forEach((item) => {
    const match = item.dataset.cview === cview && (!item.dataset.ymode || item.dataset.ymode === activeYearMode);
    item.classList.toggle('active', match);
  });

  syncTimelineFocusClass();
  updateCalViewLabel();
  updateCalendarRangeLabel();
  requestAnimationFrame(syncYearTimelineRowHeight);
}

/* ================================================================
 * Navigation handlers (prev / next / today)
 * ================================================================ */

function calendarPrev() {
  if (activeCalendarView === 'month') {
    currentMonth = (currentMonth + 11) % 12;
  } else if (activeCalendarView === 'week') {
    weekStart = addDays(weekStart, -7);
  } else if (activeCalendarView === 'agenda') {
    agendaAnchor = addDays(agendaAnchor, -7);
    agendaRangeBefore = 14; agendaRangeAfter = 14;
  }
  renderCalendar();
}

function calendarNext() {
  if (activeCalendarView === 'month') {
    currentMonth = (currentMonth + 1) % 12;
  } else if (activeCalendarView === 'week') {
    weekStart = addDays(weekStart, 7);
  } else if (activeCalendarView === 'agenda') {
    agendaAnchor = addDays(agendaAnchor, 7);
    agendaRangeBefore = 14; agendaRangeAfter = 14;
  }
  renderCalendar();
}

function calendarGoToday() {
  currentMonth = demoToday.getMonth();
  weekStart = startOfWeek(demoToday);
  agendaAnchor = new Date(demoToday);
  agendaRangeBefore = 14; agendaRangeAfter = 14;
  renderCalendar();
}

/* ================================================================
 * Calendar filter UI & initialization
 * ================================================================ */

function initCalendarUI() {
  // Account toggle
  document.querySelectorAll('button[data-account]').forEach((button) => {
    button.addEventListener('click', () => {
      activeAccount = button.dataset.account;
      document.querySelectorAll('button[data-account]').forEach((btn) => btn.classList.remove('active'));
      button.classList.add('active');
      document.getElementById('globalAccountChip').textContent = `계정: ${accountLabel[activeAccount]}`;
      renderCalendar();
    });
  });

  // View dropdown
  const calViewBtn = document.getElementById('calViewBtn');
  const calViewMenu = document.getElementById('calViewMenu');
  const calViewItems = calViewMenu.querySelectorAll('.cal-view-menu-item');

  calViewBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    calViewMenu.classList.toggle('open');
  });
  calViewItems.forEach((item) => {
    item.addEventListener('click', () => {
      switchCalendarView(item.dataset.cview, item.dataset.ymode || null);
      calViewMenu.classList.remove('open');
    });
  });
  document.addEventListener('click', () => calViewMenu.classList.remove('open'));
  calViewMenu.addEventListener('click', (e) => e.stopPropagation());

  // Todo done toggle
  document.getElementById('todoDoneToggle').addEventListener('click', () => {
    showCompletedTodos = !showCompletedTodos;
    syncTodoToggleText();
    renderCalendar();
  });

  // Navigation buttons
  document.getElementById('calendarPrev').addEventListener('click', calendarPrev);
  document.getElementById('calendarNext').addEventListener('click', calendarNext);
  document.getElementById('calendarToday').addEventListener('click', calendarGoToday);

  // Week hour range selects
  const startSel = document.getElementById('weekHourStartSelect');
  const endSel = document.getElementById('weekHourEndSelect');
  if (startSel && endSel) {
    for (let h = 0; h <= 23; h++) {
      const pad = String(h).padStart(2, '0');
      startSel.innerHTML += `<option value="${h}"${h === weekHourStart ? ' selected' : ''}>${pad}</option>`;
    }
    for (let h = 1; h <= 24; h++) {
      const label = h === 24 ? '24' : String(h).padStart(2, '0');
      endSel.innerHTML += `<option value="${h}"${h === weekHourEnd ? ' selected' : ''}>${label}</option>`;
    }
    startSel.addEventListener('change', () => {
      weekHourStart = Number(startSel.value);
      if (weekHourStart >= weekHourEnd) { weekHourEnd = weekHourStart + 1; endSel.value = weekHourEnd; }
      renderCalendar();
    });
    endSel.addEventListener('change', () => {
      weekHourEnd = Number(endSel.value);
      if (weekHourEnd <= weekHourStart) { weekHourStart = weekHourEnd - 1; startSel.value = weekHourStart; }
      renderCalendar();
    });
  }

  // Initial render
  syncTodoToggleText();
  syncTimelineFocusClass();
  updateCalViewLabel();
  renderCalendar();

  // Resize handler
  window.addEventListener('resize', () => requestAnimationFrame(syncYearTimelineRowHeight));
  window.addEventListener('load', () => requestAnimationFrame(syncYearTimelineRowHeight));
}
