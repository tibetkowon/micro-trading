# 서버 이전 — Naver Cloud Platform → Google Cloud Platform

## Goal

현재 NCP Micro(1GB RAM) 서버에서 운영 중인 자동 매매 시스템을 **Google Cloud Platform(GCP)**으로 이전한다.
DB를 Firebase Firestore로 이전(별도 계획서)하면 GCP와 동일 생태계가 되어 관리 편의성이 높아진다.

---

## 결정 사항 (Before Migration)

| 항목 | 선택지 | 권장 |
|------|--------|------|
| GCP 서비스 | Compute Engine(VM) vs Cloud Run vs App Engine | **Compute Engine e2-micro** (상시 실행, 무료 티어 1개) |
| OS | Ubuntu 22.04 LTS | Ubuntu 22.04 |
| 리전 | asia-northeast3(서울) | **asia-northeast3** (KIS API 레이턴시 최소화) |
| 배포 방식 | SSH 직접 배포 vs Docker + CI/CD | **Docker + GitHub Actions** (기존 CI/CD 재활용) |

---

## GCP e2-micro 스펙 (무료 티어)

- vCPU: 0.25 (버스트 가능)
- RAM: 1GB (현재 NCP Micro와 동일)
- 디스크: 30GB 표준 영구 디스크
- 네트워크: 월 1GB 이그레스 무료
- 비용: **$0/월** (us-central1/us-east1/us-west1 리전만 무료 — 서울은 유료)

> ⚠️ **중요**: e2-micro 무료 티어는 미국 리전(us-*)만 적용됨.
> 서울(asia-northeast3) 사용 시 약 $6~8/월 발생.
> KIS API 레이턴시 vs 비용을 고려하여 리전을 결정해야 함.

---

## Requirements

### 기능 요건
1. 기존 NCP 서버와 동일한 기능 (Go 백엔드 + React 정적 파일 서빙) 운영
2. GitHub Actions CI/CD가 GCP VM에 자동 배포
3. HTTPS 지원 (Let's Encrypt 또는 GCP Load Balancer)
4. 환경변수(`.env`) 안전하게 관리 (GCP Secret Manager 또는 VM 로컬 파일)
5. 기존 NCP 도메인/IP 교체 (DNS A 레코드 변경)

### 비기능 요건
- 다운타임 최소화: 블루-그린 방식으로 트래픽 전환 후 NCP 종료
- 1GB RAM 제약 유지 (기존 최적화 그대로 적용)

---

## Migration Steps

### Phase 0 — GCP 프로젝트 설정
1. GCP 프로젝트 생성 또는 기존 Firebase 프로젝트와 통합
2. Compute Engine API 활성화
3. e2-micro VM 생성 (asia-northeast3, Ubuntu 22.04)
4. 고정 외부 IP 할당
5. 방화벽 규칙: TCP 80, 443, 22 허용

### Phase 1 — VM 환경 세팅
```bash
# Docker 설치
sudo apt-get update && sudo apt-get install -y docker.io
sudo systemctl enable docker

# GitHub Actions 배포용 서비스 계정 또는 SSH 키 등록
```

### Phase 2 — CI/CD 파이프라인 수정 (GitHub Actions)
현재 `.github/workflows/` 수정:
- `SSH_HOST`: NCP IP → GCP IP (GitHub Secret 변경)
- `SSH_USER`: 변경 없거나 `ubuntu`로 조정
- Docker 이미지 빌드 및 VM SSH 배포 방식 유지

### Phase 3 — 환경변수 마이그레이션
방법 A (단순): `.env` 파일을 VM에 SCP로 복사  
방법 B (권장): GCP Secret Manager에 저장, 앱 시작 시 `gcloud secrets versions access` 로 로드

```bash
# Secret Manager에 .env 내용 저장
gcloud secrets create trading-env --data-file=.env
```

### Phase 4 — 데이터 마이그레이션
Firebase Firestore 마이그레이션 완료 후 이 단계 불필요.
SQLite 파일만 사용 중이라면: `scp trading.db gcp-vm:/app/trading.db`

### Phase 5 — 도메인 전환
1. GCP VM에서 서비스 정상 동작 확인
2. DNS A 레코드를 GCP 외부 IP로 변경
3. Let's Encrypt 인증서 재발급 (certbot)
4. NCP 서버 1주일 대기 후 종료

---

## Affected Files (코드 변경)

| 파일 | 변경 내용 |
|------|---------|
| `.github/workflows/deploy.yml` | SSH_HOST Secret 변경, GCP 관련 설정 조정 |
| `Dockerfile` (있으면) | 변경 없음 |
| `backend/.env.example` | `FIREBASE_CREDENTIALS_JSON` 추가 (Phase 2 DB 마이그레이션 연계) |

---

## 체크리스트 (이전 완료 기준)

- [ ] GCP VM에서 `go build ./...` + `npm run build` 성공
- [ ] 트레이딩 엔진 09:00~15:30 스케줄 정상 동작
- [ ] KIS WebSocket 연결 안정성 확인 (GCP ↔ KIS API 레이턴시)
- [ ] GitHub Actions 자동 배포 성공
- [ ] HTTPS 접속 가능
- [ ] NCP 서버 종료

---

## 위험 및 대응

| 위험 | 대응 |
|------|------|
| GCP-KIS API 레이턴시 증가 | 서울 리전 사용, WebSocket 안정성 모니터링 |
| 무료 티어 비용 (서울 리전 시 ~$8/월) | 미국 리전도 검토 (레이턴시 허용 가능하면) |
| 배포 중 거래 중단 | 장중(09:00~15:30) 외 배포 |
| 환경변수 유출 | GCP Secret Manager 사용 권장 |
