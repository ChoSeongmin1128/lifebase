# Google 연동 규칙

## 색상
- Google Calendar 색상(이벤트 11종, 캘린더 24종)을 하드코딩하지 않는다
- 반드시 colors.get() API로 런타임에 hex 값을 조회한다
- Classic/Modern 팔레트 차이가 있으므로 코드에 hex 값을 직접 쓰지 않는다

## 동기화 전략
- Calendar: 웹훅(events.watch()) + syncToken 증분 동기화
- Google Tasks: 폴링(tasks.list() + updatedMin) — 웹훅 미지원
- 두 도메인의 동기화 워커를 별도로 설계한다

## Google Tasks 확장 필드
- Google Tasks에 없는 기능(우선순위, 시간, 반복, 색상)은 LifeBase DB에만 저장한다
- Google Tasks 동기화 대상: title, notes, due(날짜만), status, parent
- 확장 필드는 LifeBase → Google 방향으로 동기화하지 않는다

## 쿼터
- Calendar: 일일 1,000,000 쿼리
- Tasks: 일일 50,000 쿼리 (Calendar의 1/20)
- Tasks 폴링 빈도를 보수적으로 설계한다
