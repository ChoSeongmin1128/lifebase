/**
 * LifeBase Prototype — Shared Demo Data
 *
 * 프로토타입 HTML 파일(web, desktop, mobile)에서 공통으로 사용하는
 * 상수와 데모 데이터를 한 곳에서 관리한다.
 * <script src="shared/demo-data.js"> 로 로드하면 전역 변수로 노출된다.
 */

const YEAR = 2026;

const DOW_KR = ['일', '월', '화', '수', '목', '금', '토'];

const demoToday = new Date(YEAR, 1, 21);

const accountLabel = {
  all: '전체',
  personal: 'personal@gmail.com',
  work: 'team.ops@google.com'
};

const holidays = {
  '2026-01-01': '신정',
  '2026-03-01': '삼일절',
  '2026-05-05': '어린이날',
  '2026-06-06': '현충일',
  '2026-08-15': '광복절',
  '2026-10-03': '개천절',
  '2026-10-09': '한글날',
  '2026-12-25': '성탄절'
};

const rangeEvents = [
  { start: '2026-02-18', end: '2026-02-21', title: '릴리즈 안정화', color: 'red', account: 'work' },
  { start: '2026-02-17', end: '2026-02-23', title: '장애 대응', color: 'yellow', account: 'work' },
  { start: '2026-02-15', end: '2026-02-20', title: '출장', color: 'lavender', account: 'personal' },
  { start: '2026-03-22', end: '2026-03-26', title: '해외 컨퍼런스', color: 'blue', account: 'work' },
  { start: '2026-06-28', end: '2026-07-03', title: '프로젝트 마감', color: 'green', account: 'personal' }
];

const singleEvents = [
  { date: '2026-02-18', title: '스프린트 점검', color: 'teal', account: 'work' },
  { date: '2026-02-19', title: '워크숍', color: 'blue', account: 'work' },
  { date: '2026-02-23', title: '긴급 미팅', color: 'red', account: 'work' },
  { date: '2026-03-10', title: '고객 리뷰', color: 'yellow', account: 'work' },
  { date: '2026-05-11', title: '어머니날', color: 'lavender', account: 'personal' }
];

const timedEvents = [
  { date: '2026-02-16', time: '09:00', duration: 1, title: '스프린트 점검', color: 'teal', account: 'work' },
  { date: '2026-02-16', time: '14:00', duration: 1.5, title: '1:1 미팅', color: 'lavender', account: 'work' },
  { date: '2026-02-18', time: '10:00', duration: 2, title: '디자인 리뷰', color: 'blue', account: 'work' },
  { date: '2026-02-18', time: '14:00', duration: 1, title: '코드 리뷰', color: 'teal', account: 'work' },
  { date: '2026-02-19', time: '13:00', duration: 3, title: '워크숍', color: 'yellow', account: 'work' },
  { date: '2026-02-19', time: '09:30', duration: 1, title: '스탠드업', color: 'green', account: 'work' },
  { date: '2026-02-20', time: '09:00', duration: 1.5, title: '가족 일정', color: 'green', account: 'personal' },
  { date: '2026-02-20', time: '15:00', duration: 1, title: '치과 예약', color: 'lavender', account: 'personal' },
  { date: '2026-02-21', time: '11:00', duration: 1, title: '점심 약속', color: 'green', account: 'personal' },
  { date: '2026-02-22', time: '18:00', duration: 1.5, title: '장애 대응 회의', color: 'red', account: 'work' }
];

const todoItems = [
  { id: 1, date: '2026-02-18', title: '릴리즈 체크리스트 점검', done: false, account: 'work' },
  { id: 2, date: '2026-02-20', title: '회의 노트 정리', done: true, account: 'work' },
  { id: 3, date: '2026-02-24', title: '월간 회고 작성', done: false, account: 'work' },
  { id: 4, date: '2026-03-15', title: 'OAuth 토큰 점검', done: false, account: 'work' },
  { id: 5, date: '2026-04-18', title: '세금 신고 자료 업로드', done: false, account: 'personal' }
];

const galleryItems = [
  { id: 1, name: 'launch-day.jpg', type: 'image', created: '2026-02-22', modified: '2026-02-22' },
  { id: 2, name: 'office-tour.mp4', type: 'video', created: '2026-02-21', modified: '2026-02-21' },
  { id: 3, name: 'meeting-shot.png', type: 'image', created: '2026-02-20', modified: '2026-02-20' },
  { id: 4, name: 'demo-record.mov', type: 'video', created: '2026-02-18', modified: '2026-02-18' },
  { id: 5, name: 'whiteboard.jpg', type: 'image', created: '2026-02-16', modified: '2026-02-16' },
  { id: 6, name: 'family-trip.mp4', type: 'video', created: '2026-01-19', modified: '2026-01-19' },
  { id: 7, name: 'sunset-walk.jpg', type: 'image', created: '2026-01-19', modified: '2026-01-19' },
  { id: 8, name: 'team-photo.png', type: 'image', created: '2026-01-15', modified: '2026-01-15' }
];
