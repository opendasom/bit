<p align="center">
  <img src="public/bit-logo.png" alt="Bit logo" width="300" />
</p>

# Bit

IPFS와 이더리움 위에서 동작하는 실험적 분산 버전 관리 프로토콜입니다. 커밋 diff와 메타데이터는 IPFS에, 브랜치·커밋 상태는 `BitRegistry` 스마트 컨트랙트에 저장합니다.

> **Status: Alpha.** 프로토콜과 저장 형식은 변경될 수 있으며, 프로덕션 사용이나 실제 자산이 걸린 환경을 위한 보안 감사를 받지 않았습니다.

중앙 Git 서버 없이 코드 히스토리를 content-addressed 형태로 저장하고 누구나 검증할 수 있습니다. 온체인 레코드는 히스토리의 무결성을 검증할 수 있게 하지만, IPFS 데이터의 가용성은 핀(pin)과 복제 상태에 달려 있습니다. 중요한 데이터는 자체 IPFS 노드 또는 신뢰할 수 있는 pinning 서비스에 보관하세요.

---

## 영상

1. Maintainer Creating Repository  
[![Video Label](http://img.youtube.com/vi/f56hsj97zWA/0.jpg)](https://youtu.be/f56hsj97zWA)

2. Contributor Developing  
[![Video Label](http://img.youtube.com/vi/HiM1U8WBm3g/0.jpg)](https://youtu.be/HiM1U8WBm3g)

3. Maintainer Approving PR  
[![Video Label](http://img.youtube.com/vi/qIJjybwHhjo/0.jpg)](https://youtu.be/qIJjybwHhjo)

---

## 설치

**사전 조건**

- Go 1.25+ (`go.mod`의 toolchain 설정이 검증된 Go 버전을 선택합니다)
- Node.js 20.19+ 및 npm
- Foundry 1.7.1+ (`anvil`, `forge`) — 로컬 체인 및 컨트랙트 배포용
- IPFS 데몬 (Kubo) — `ipfs daemon`
- 이더리움 노드 접근 (로컬: Anvil, 테스트넷: Sepolia 등)

```bash
git clone https://github.com/opendasom/bit.git
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

**터미널 3 — 프로젝트 루트에서 컨트랙트 배포**
```bash
forge create --broadcast \
  --rpc-url http://127.0.0.1:8545 \
  --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
  contracts/src/BitRegistry.sol:BitRegistry
# 출력의 "Deployed to: 0x..." 주소를 --contract 플래그에 사용
```

빈 기본 Anvil 인스턴스의 첫 배포 주소는 `0x5FbDB2315678afecb367f032d93F642f64180aa3`입니다.
로컬 IPFS 데몬과 새 BitRegistry가 실행 중일 때 데모 저장소, 브랜치, 커밋, PR 상태를 추가할 수 있습니다.

```bash
# 저장소 루트에서 실행
cd ..
npm run anvil:seed
```

시드 명령은 기본 Anvil 계정 4개를 사용하며 비어 있는 registry에서 한 번만 실행합니다.
README에 나온 private key는 Anvil의 공개된 **로컬 테스트 전용** 키입니다. 어떤 테스트넷이나 메인넷에서도 사용하면 안 됩니다.

---

## 빠른 시작

### 1. 저장소 초기화

```bash
mkdir my-project && cd my-project
git init

export BIT_PRIVATE_KEY=ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
bit init \
  --rpc http://127.0.0.1:8545 \
  --contract 0xYourContractAddress \
  --name my-project
# --ipfs 생략 시 기본값: http://localhost:5001
```

쓰기 명령 전에는 개인키를 config나 shell history에 저장하지 않고 환경변수로 제공합니다.

성공하면 `.bit/config.json`이 생성되고 체인에 저장소가 등록됩니다 (repoId 발급).

### 2. remote 추가

```bash
# URL 형식: bit://<network>/<contractAddress>/<repoId>
bit remote add origin bit://local/0xYourContractAddress/1
```

현재 CLI는 remote URL에서 `repoId`를 읽고, RPC URL과 컨트랙트 주소는 `bit init`으로 만든 `.bit/config.json`을 사용합니다. URL의 network·contract 값과 로컬 설정 값은 같은 배포 환경을 가리키도록 맞추세요.

### 3. push

```bash
git add . && git commit -m "first commit"
bit push origin
# 현재 브랜치를 자동 감지해서 push
```

> merge commit push는 지원하지 않습니다 (linear history only).

### 4. pull

다른 머신이나 디렉토리에서 코드를 받을 때는 read-only `clone`을 사용합니다. 개인키나 새 on-chain repository 생성이 필요하지 않습니다.

```bash
bit clone bit://local/0xYourContractAddress/1 other-project \
  --rpc http://127.0.0.1:8545 \
  --ipfs http://127.0.0.1:5001 \
  --branch main
```

### 5. Fork 및 PR 생성/관리 (웹)

웹 explorer에서 현재 브랜치를 새 온체인 저장소로 fork하거나 PR을 생성/승인/거부/닫을 수 있습니다.

- **Fork `<branch>`** 버튼은 현재 브랜치의 온체인 커밋 히스토리와 IPFS CID 포인터를 하나의 atomic transaction으로 복제합니다.
- fork와 PR은 한 작업당 최대 64개 커밋을 지원하여 block gas 초과를 방지합니다.
- fork와 PR 생성/승인/거부/닫기는 모두 웹 explorer에서 MetaMask로 서명합니다.

1. 웹에서 원본 저장소와 브랜치를 선택하고 **Fork**를 실행합니다.
2. 생성된 fork의 `bit://` URL을 사용해 `bit clone <fork-url> my-fork`로 로컬 작업 사본을 만듭니다.
3. 로컬에서 새 커밋을 만들고 `BIT_PRIVATE_KEY=... bit push origin`으로 fork에 push합니다.
4. 웹에서 fork의 **Pull Requests** 탭을 열고 대상 저장소/브랜치, 소스 브랜치, 설명을 입력합니다.
5. 대상 저장소 Maintainer는 **Approve**(fast-forward 반영) / **Reject**할 수 있고, 작성자 또는 Maintainer는 **Close**할 수 있습니다.

- target 브랜치가 fork 시점 이후 먼저 앞서 나갔다면 생성이 거절됩니다.
- 현재 브랜치에 새 커밋이 없으면 PR 생성이 거절됩니다.
- 실제 반영은 `approve` 시점에 다시 검증됩니다.

> **주의**: indexed PR range와 atomic fork가 추가되어 ABI가 변경되었습니다. 기존 배포와 호환되지 않습니다. PR 설명은 체인에 저장됩니다. 새 BitRegistry를 배포했다면 `.env.local`의 `VITE_BIT_CONTRACT`를 새 주소로 변경하고 웹 개발 서버를 다시 시작하세요.

### 6. 웹 explorer

현재 체인에 등록된 저장소 목록과 각 저장소의 커밋 메타데이터를 읽기 전용으로 확인할 수 있습니다.

```bash
# 저장소 루트에서 실행
cp .env.example .env.local
# .env.local의 RPC URL, 체인 ID, 컨트랙트 주소, IPFS gateway를 환경에 맞게 수정
npm ci
npm run web:dev
```

로컬 연결값은 Git에 포함되지 않는 `.env.local`에 둡니다.

```dotenv
VITE_BIT_CHAIN_ID=31337
VITE_BIT_RPC_URL=http://127.0.0.1:8545
VITE_BIT_CONTRACT=0xYourContractAddress
VITE_BIT_IPFS_API=/ipfs-api
VITE_BIT_IPFS_GATEWAY=http://127.0.0.1:8080/ipfs
```

브라우저의 **Connection** 메뉴에서도 RPC URL, Contract, IPFS Gateway를 바꾸고 새로고침할 수 있습니다.

- RPC URL: Anvil 또는 테스트넷 RPC URL
- Contract: 배포된 `BitRegistry` 컨트랙트 주소
- IPFS Gateway: 예: `http://127.0.0.1:8080/ipfs`
- Chain ID: Anvil은 `31337`, Sepolia는 `11155111`
- Branch: repository metadata의 default branch, 없으면 `main`

웹 번들에는 private key를 넣지 않습니다.
웹은 대부분의 데이터를 읽기용 gateway에서 가져옵니다. Fork 이름을 변경하면 새 repository metadata를 로컬 IPFS API에 업로드하며,
개발 서버는 `/ipfs-api`를 `http://127.0.0.1:5001`로 프록시합니다.
체인에 쓰는 fork, role 변경, PR 생성/승인/거부/닫기 트랜잭션은 MetaMask로 서명합니다.

웹 화면은 커밋 메시지, 작성자, 작성일, 온체인 기록자, 온체인 기록 시간, 부모 커밋만 표시합니다. diff 내용은 표시하지 않습니다.
다만 현재 diff CID는 온체인/IPFS에 공개되어 있으므로, 이는 UI 제한입니다. 코드 diff 자체를 비공개로 만들려면 diff 암호화와 권한별 복호화 키 관리가 추가로 필요합니다.

---

## 명령어 참조

### `bit init`

```
BIT_PRIVATE_KEY=<privkey> bit init --rpc <url> --contract <addr> [--ipfs <url>] [--name <repo-name>] [--description <text>] [--branch <branch>]
```

- `.git`이 없으면 에러. 먼저 `git init` 필요.
- 저장소 metadata를 IPFS에 업로드한 뒤 체인에 저장소를 생성하고 `.bit/config.json`을 저장합니다.
- 개인키는 `.bit/config.json`에 저장하지 않으며 `.bit/`는 `.git/info/exclude`에 자동 등록됩니다.
- `--key`는 하위 호환용 deprecated 옵션입니다. `BIT_PRIVATE_KEY` 환경변수를 사용하세요.
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
- 모든 manifest/diff/parent를 먼저 검증하고 Git object를 격리된 index에서 재구성한 뒤 branch를 한 번만 전환합니다.
- dirty worktree에서는 실행을 거절합니다.

### `bit clone`

```
bit clone <bit-url> [directory] --rpc <url> [--ipfs <url>] [--branch <branch>]
```

- private key 없이 기존 repository를 복원합니다.
- `origin` remote와 local config를 생성한 뒤 검증된 branch를 checkout합니다.

---

## 권한 모델 (BitRegistry)

| Role | 권한 |
|------|------|
| Owner | 웹 또는 setRole로 다른 사용자 역할 지정; 마지막 Owner 제거는 금지 |
| Maintainer | push(recordCommit), PR 승인/거부 (웹에서 처리) |
| Contributor | 명시적 참여자 표시 (쓰기 권한 없음) |
| None | 조회만 가능 |

저장소 생성자는 Owner가 되며, Owner는 Maintainer 권한을 포함합니다.

---

## 현재 제약과 보안 경계

- push와 merge는 linear history만 지원하며 merge commit은 거절됩니다.
- atomic fork와 단일 PR은 최대 64개 커밋을 처리합니다.
- 웹 commit 화면은 선택한 브랜치의 최근 50개 커밋을 표시합니다.
- IPFS 데이터는 공개되어 있으며 암호화되지 않습니다. CID를 아는 사용자는 diff와 metadata를 읽을 수 있습니다.
- CID의 지속적인 가용성은 하나 이상의 IPFS 노드가 데이터를 pin하고 제공하는지에 달려 있습니다.
- protocol v2 client는 이전 BitRegistry 배포와 호환되지 않으며 연결 시 `PROTOCOL_VERSION`을 확인합니다.

---

## 검증

```bash
go test ./...
go vet ./...
forge test
npm run typecheck
npm run test:web
npm run web:build
npm run format:check
```

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
│   └── clone.go      # bit clone (read-only bootstrap)
├── internal/
│   ├── app/          # 명령 실행 로직 (cmd는 얇은 래퍼)
│   ├── chain/        # BitRegistry 컨트랙트 연동 (go-ethereum)
│   ├── git/          # .git 읽기/쓰기 (go-git + exec git)
│   ├── ipfs/         # IPFS HTTP API 클라이언트
│   ├── cid/          # CIDv0 ↔ bytes32 변환 (외부 의존성 없음)
│   ├── manifest/     # manifest JSON 인코딩/디코딩
│   └── config/       # .bit/config.json 관리
├── contracts/
│   └── src/BitRegistry.sol   # Solidity 컨트랙트
├── web/                      # React explorer 및 MetaMask workflow
└── scripts/                  # 로컬 Anvil 시드 데이터
```

---

## 주요 의존성

| 패키지 | 용도 | 라이선스 |
|--------|------|----------|
| `go-ethereum v1.17.0` | 이더리움 클라이언트, ABI 바인딩 | LGPL-3.0 (library) |
| `go-git/v5 v5.19.1` | Git 저장소 읽기 | Apache-2.0 |
| `cobra v1.8.1` | CLI 프레임워크 | Apache-2.0 |
| `golang.org/x/crypto v0.51.0` | 암호화 유틸리티 | BSD-3-Clause |
| `react`, `react-dom`, `viem` | 웹 explorer 런타임 | MIT |
| `typescript` | 웹 빌드 도구 | Apache-2.0 |

정확한 적용 범위와 릴리스 시 유의사항은 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)를 확인하세요.

---

## 개발과 검증

```bash
go test ./...
go vet ./...

cd contracts && forge test && cd ..

npm ci
npm run web:build
```

기여 방법은 [CONTRIBUTING.md](CONTRIBUTING.md), 취약점 신고는 [SECURITY.md](SECURITY.md)를 참고하세요.

---

## License

이 저장소에서 작성한 Bit 소스 코드는 [MIT License](LICENSE)로 배포됩니다. 제3자 컴포넌트는 각자의 라이선스를 유지하며, 특히 `go-ethereum`을 링크한 CLI 바이너리를 배포할 때는 LGPL-3.0 의무를 함께 충족해야 합니다. 자세한 내용은 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)를 확인하세요.
