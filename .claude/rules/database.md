# 데이터베이스 컨벤션

## 마이그레이션
- 도구: goose (Go 코드 내 실행 가능, NO TRANSACTION 옵션 지원)
- 마이그레이션 파일은 `migrations/` 디렉토리에 순차 번호로 관리
- 스키마 변경은 반드시 Up/Down 쌍으로 작성한다

## 네이밍
- 테이블명: snake_case, 복수형 (예: `users`, `todo_lists`, `event_reminders`)
- 컬럼명: snake_case (예: `created_at`, `user_id`, `is_pinned`)
- 참조 컬럼: `_id` 접미사로 관계 명시 (예: `user_id`, `folder_id`)

## FK 정책
- DB에 Foreign Key 제약을 걸지 않는다
- 참조 무결성은 코드(UseCase/도메인 레이어)에서 검증한다
- 이유: 마이그레이션, 벌크 삭제, 운영 작업 시 FK 의존 순서로 인한 복잡성 제거

## 논리적 관계 표기
- DB 스키마 문서/ERD에는 논리적 관계(logical FK)를 반드시 표기한다

## 인덱스
- FK를 걸지 않으므로 참조 컬럼의 인덱스를 수동으로 생성한다
- JOIN/조회 성능을 위해 `_id` 컬럼에는 인덱스를 기본으로 건다

## 타임스탬프
- 모든 테이블에 `created_at`, `updated_at` 컬럼 필수 (timestamptz, UTC)
- Soft delete 대상 테이블은 `deleted_at` 컬럼 추가 (nullable timestamptz)

## Soft Delete
- 휴지통 대상(files, folders): `deleted_at` 기반 soft delete
- 30일 경과 후 워커가 hard delete (물리 파일 -> 썸네일 -> DB 순서)

## 마이그레이션 체크리스트
- 변경 전: 영향 테이블/인덱스/쿼리 경로를 기록한다
- 변경 중: 인덱스/기본값/null 허용 여부를 명시한다
- 변경 후: `migrate:up`, `migrate:status`, 핵심 조회 경로를 검증한다
- 롤백 기준: 운영 리스크가 있으면 즉시 `migrate:down` 가능해야 한다
