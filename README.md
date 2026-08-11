# bit

IPFS와 이더리움 블록체인 위에서 동작하는 탈중앙화 버전 관리 시스템(DVCS).

커밋 diff와 메타데이터는 IPFS에, 브랜치/커밋 상태는 스마트 컨트랙트(BitRegistry)에 저장됩니다.
중앙 서버 없이 코드 히스토리를 영구적으로 보존하고 누구나 검증할 수 있습니다.

---
## 영상

1. Maintainer Creating Repository  
[![Video Label](http://img.youtube.com/vi/f56hsj97zWA/0.jpg)](https://youtu.be/f56hsj97zWA)

2. Contributor Developing  
[![Video Label](http://img.youtube.com/vi/HiM1U8WBm3g/0.jpg)](https://youtu.be/HiM1U8WBm3g)

3. Maintainer Approving PR  
[![Video Label](http://img.youtube.com/vi/qIJjybwHhjo/0.jpg)](https://youtu.be/qIJjybwHhjo)

---

---

## 설치

**사전 조건**

- Go 1.25.0+
- IPFS 데몬 (Kubo) — `ipfs daemon`
- 이더리움 노드 접근 (로컬: Anvil, 테스트넷: Sepolia 등)

```bash
git clone https://github.com/hsh-719/bit.git
cd bit
go build -o bit .

# 전역 명령어로 등록 (선택)
sudo cp ./bit /usr/local/bin/bit
```

---

## 로컬 테스트 환경 구성

터미널 3개를 열어 순서대로 실행합니다.

**터미널 1 — Anvil 실행**
```bash
anvil
# 출력에서 private key 복사 (기본 첫 번째 키 사용)
```

**터미널 2 — IPFS 데몬 실행**
```bash
ipfs daemon
```

**터미널 3 — 컨트랙트 배포**
```bash
cd contracts
forge create --broadcast \
  --rpc-url http://127.0.0.1:8545 \
  --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
  src/BitRegistry.sol:BitRegistry
# 출력의 "Deployed to: 0x..." 주소를 --contract 플래그에 사용
```

---

## 빠른 시작

### 1. 저장소 초기화

```bash
mkdir my-project && cd my-project
git init

bit init \
  --rpc http://127.0.0.1:8545 \
  --contract 0xYourContractAddress \
  --key ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
# --ipfs 생략 시 기본값: http://localhost:5001
```

성공하면 `.bit/config.json`이 생성되고 체인에 저장소가 등록됩니다 (repoId 발급).

### 2. remote 추가

```bash
# URL 형식: bit://<network>/<contractAddress>/<repoId>
bit remote add origin bit://local/0xYourContractAddress/1
```

### 3. push

```bash
git add . && git commit -m "first commit"
bit push origin
# 현재 브랜치를 자동 감지해서 push
```

> merge commit push는 지원하지 않습니다 (linear history only).

### 4. pull

다른 머신이나 디렉토리에서 코드를 받아옵니다.

```bash
mkdir other-project && cd other-project
git init

bit init \
  --rpc http://127.0.0.1:8545 \
  --contract 0xYourContractAddress \
  --key YourPrivateKeyHex

bit remote add origin bit://local/0xYourContractAddress/1
bit pull origin main
```

### 5. fork

다른 사람의 저장소를 fork해서 내 저장소로 만듭니다.
빈 디렉토리에서 실행하면 `git init`까지 자동으로 처리됩니다.

```bash
mkdir my-fork && cd my-fork

bit fork bit://local/0xContractAddress/1 \
  --rpc http://127.0.0.1:8545 \
  --contract 0xYourContractAddress \
  --key YourPrivateKeyHex
# --branch 생략 시 기본값: main
```

fork 완료 후 자동으로 설정됩니다:
- `origin` → 내 fork (새로 생성된 저장소)
- `upstream` → 원본 저장소

이후 새 커밋을 만들고 `bit push origin`으로 내 fork에 push할 수 있습니다.

### 5-1. Fork 및 PR 생성/관리 (웹)

웹 explorer에서 현재 브랜치를 새 온체인 저장소로 fork하거나 PR을 생성/승인/거부/닫을 수 있습니다.

- **Fork `<branch>`** 버튼은 현재 브랜치의 온체인 커밋 히스토리와 IPFS CID 포인터를 새 저장소로 복제합니다. 커밋 수만큼 MetaMask 트랜잭션 승인이 필요합니다.
- 웹 fork는 브라우저에 로컬 Git 워킹트리를 만들지 않습니다. 로컬 파일을 포함한 fork가 필요하면 기존 `bit fork` 명령을 사용합니다.

1. fork 저장소에 새 커밋을 만들고 `bit push origin`으로 push합니다.
2. 웹에서 **Connect MetaMask** 후 해당 저장소의 **Pull Requests** 탭을 엽니다.
3. **New pull request** 버튼으로 대상 저장소/브랜치, 소스 브랜치, 설명을 입력하고 제출합니다.
4. 대상 저장소의 Maintainer는 웹에서 **Approve**(fast-forward 반영) / **Reject**할 수 있고, 작성자 또는 Maintainer는 **Close**할 수 있습니다.

- target 브랜치가 fork 시점 이후 먼저 앞서 나갔다면 생성이 거절됩니다.
- 현재 브랜치에 새 커밋이 없으면 PR 생성이 거절됩니다.
- 실제 반영은 `approve` 시점에 다시 검증됩니다.

> **주의**: PR 설명은 체인에 저장되므로 `createPullRequest` 시그니처가 변경되었습니다. 웹에서 PR 기능을 쓰려면 BitRegistry를 새로 배포한 뒤 웹의 contract 주소(`web/src/main.tsx`의 `defaultContract`)를 갱신해야 합니다.

### 6. 웹 explorer

현재 체인에 등록된 저장소 목록과 각 저장소의 커밋 메타데이터를 읽기 전용으로 확인할 수 있습니다.

```bash
npm install
npm run web:dev
```

브라우저에서 표시되는 URL로 접속한 뒤 다음 값을 입력합니다.

- RPC URL: Anvil 또는 테스트넷 RPC URL
- Contract: 배포된 `BitRegistry` 컨트랙트 주소
- IPFS Gateway: 예: `http://127.0.0.1:8080/ipfs`
- Branch: 기본값 `main`

웹은 공개 설정값을 코드에 하드코딩합니다. private key는 넣지 않습니다.
웹은 IPFS API를 사용하지 않고, 읽기용 gateway만 사용합니다.
체인에 쓰는 트랜잭션(PR 생성/승인/거부/닫기)은 MetaMask로 서명합니다.

웹 화면은 커밋 메시지, 작성자, 작성일, 온체인 기록자, 온체인 기록 시간, 부모 커밋만 표시합니다. diff 내용은 표시하지 않습니다.
다만 현재 diff CID는 온체인/IPFS에 공개되어 있으므로, 이는 UI 제한입니다. 코드 diff 자체를 비공개로 만들려면 diff 암호화와 권한별 복호화 키 관리가 추가로 필요합니다.

---

## 명령어 참조

### `bit init`

```
bit init --rpc <url> --contract <addr> --key <privkey> [--ipfs <url>] [--name <repo-name>] [--description <text>] [--branch <branch>]
```

- `.git`이 없으면 에러. 먼저 `git init` 필요.
- 저장소 metadata를 IPFS에 업로드한 뒤 체인에 저장소를 생성하고 `.bit/config.json`을 저장합니다.
- `--name` 생략 시 현재 디렉토리명이 웹 표시 이름으로 사용됩니다.
- `--branch` 생략 시 `main`이 웹 기본 브랜치로 사용됩니다.

### `bit remote add`

```
bit remote add <name> <url>
```

- URL 형식: `bit://<network>/<contractAddress>/<repoId>`

### `bit push`

```
bit push <remote>
```

- 현재 브랜치를 자동 감지합니다 (인자 없음).
- 체인의 현재 헤드 이후 커밋들을 순서대로 push합니다.
- 각 커밋마다 diff와 manifest를 IPFS에 업로드하고 체인에 기록합니다.

### `bit pull`

```
bit pull <remote> <branch>
```

- 로컬 HEAD가 원격 히스토리에 없으면 에러 (diverged).
- 누락 커밋을 IPFS에서 받아 원본 커밋을 완전히 재구성합니다 (커밋 hash 보존).

### `bit fork`

```
bit fork <bitURL> [--rpc <url>] [--contract <addr>] [--key <key>] [--ipfs <url>] [--branch <branch>]
```

- `.git`이 없으면 자동으로 `git init`을 실행합니다.
- `--rpc/--contract/--key` 생략 시 기존 `.bit/config.json`에서 읽습니다.
- 원본 저장소(A)의 IPFS diff를 그대로 참조하므로 IPFS 재업로드가 없습니다.
- fork된 저장소(B)의 체인에는 A와 동일한 IPFS digest 포인터가 기록됩니다.

---

## 권한 모델 (BitRegistry)

| Role | 권한 |
|------|------|
| Owner | setRole로 다른 사용자 역할 지정 |
| Maintainer | push(recordCommit), PR 승인/거부 (웹에서 처리) |
| Contributor | 역할 없음 (현재 미사용) |
| None | 조회만 가능 |

저장소 생성자는 자동으로 Owner + Maintainer가 됩니다.

---

## 프로젝트 구조

```
bit/
├── main.go
├── cmd/
│   ├── root.go       # 루트 커맨드, 서브커맨드 등록
│   ├── init.go       # bit init
│   ├── remote.go     # bit remote add
│   ├── push.go       # bit push
│   ├── pull.go       # bit pull
│   └── fork.go       # bit fork (pr 명령은 제거됨 — PR은 웹에서 처리)
├── internal/
│   ├── app/          # 명령 실행 로직 (cmd는 얇은 래퍼)
│   ├── chain/        # BitRegistry 컨트랙트 연동 (go-ethereum)
│   ├── git/          # .git 읽기/쓰기 (go-git + exec git)
│   ├── ipfs/         # IPFS HTTP API 클라이언트
│   ├── cid/          # CIDv0 ↔ bytes32 변환 (외부 의존성 없음)
│   ├── manifest/     # manifest JSON 인코딩/디코딩
│   └── config/       # .bit/config.json 관리
└── contracts/
    └── src/BitRegistry.sol   # Solidity 컨트랙트
```

---

## 의존성

| 패키지 | 용도 |
|--------|------|
| `go-ethereum v1.13.14` | 이더리움 클라이언트, ABI 바인딩 |
| `go-git/v5 v5.19.1` | git 저장소 읽기 |
| `cobra v1.8.0` | CLI 프레임워크 |
| `golang.org/x/crypto v0.50.0` | 암호화 유틸리티 |
