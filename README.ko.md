<p align="right">
  <a href="README.md">English</a> · <strong>한국어</strong>
</p>

<p align="center">
  <img src="docs/assets/bit-logo-readme.png" alt="Bit logo" width="300" />
</p>

<h1 align="center">Bit</h1>

<p align="center">
  <strong>An open protocol for verifiable source history.</strong><br />
  Git-compatible workflows, content-addressed data on IPFS, and repository state verified on Ethereum.
</p>

<p align="center">
  <a href="https://github.com/opendasom/bit/actions/workflows/ci.yml"><img src="https://github.com/opendasom/bit/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI status" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-7cd39c?style=flat-square" alt="MIT License" /></a>
  <img src="https://img.shields.io/badge/status-alpha-f3ae84?style=flat-square" alt="Alpha status" />
</p>

<p align="center">
  <a href="#빠른-시작">Quick start</a> ·
  <a href="#데모">Demos</a> ·
  <a href="#명령어-참조">CLI reference</a> ·
  <a href="CONTRIBUTING.md">Contributing</a> ·
  <a href="SECURITY.md">Security</a>
</p>

IPFS와 이더리움 위에서 동작하는 실험적 분산 버전 관리 프로토콜입니다. 커밋 diff와 메타데이터는 IPFS에, 브랜치·커밋 상태는 `BitRegistry` 스마트 컨트랙트에 저장합니다.

> [!WARNING]
> **Alpha software.** 프로토콜과 저장 형식은 변경될 수 있으며, 프로덕션 사용이나 실제 자산이 걸린 환경을 위한 보안 감사를 받지 않았습니다.

중앙 Git 서버 없이 코드 히스토리를 content-addressed 형태로 저장하고 누구나 검증할 수 있습니다. 온체인 레코드는 히스토리의 무결성을 검증할 수 있게 하지만, IPFS 데이터의 가용성은 핀(pin)과 복제 상태에 달려 있습니다. 중요한 데이터는 자체 IPFS 노드 또는 신뢰할 수 있는 pinning 서비스에 보관하세요.

## 핵심 구성

| 계층 | Bit가 하는 일 |
|---|---|
| **Git client** | 익숙한 Git 저장소와 커밋을 로컬에서 생성하고, `bit` CLI로 원격 상태와 동기화합니다. |
| **IPFS** | diff, manifest, repository metadata를 content-addressed 객체로 저장하고 복제합니다. |
| **Ethereum** | `BitRegistry`가 저장소 상태, Git 커밋 해시 기반 브랜치 HEAD, Manifest·Diff CID digest 및 사용자 역할을 기록합니다. |
| **Web explorer** | [`bit-w3`](https://github.com/opendasom/bit-w3)에서 IPFS와 체인 상태를 읽고, MetaMask 서명으로 fork·역할·PR 작업을 수행합니다. |

<p align="center">
  <img src="docs/assets/bit-architecture.png" alt="참여자, 로컬 Git·CLI 환경, IPFS, Ethereum 및 Web3 클라이언트의 연결을 보여 주는 Bit 아키텍처" width="100%" />
</p>

### 웹 explorer

배포된 [Bit Web3 explorer](https://bitweb.space/)에서 저장소를 탐색하고 레지스트리 작업에 서명할 수 있습니다.

<table>
  <tr>
    <td width="50%"><img src="docs/assets/web-explorer-home.png" alt="Ethereum 연결 제어와 MetaMask 연결 버튼이 있는 Bit Web3 explorer 랜딩 화면" /></td>
    <td width="50%"><img src="docs/assets/web-explorer-pull-requests.png" alt="Pull Request와 서명 권한 정보를 보여 주는 Bit Web3 explorer 화면" /></td>
  </tr>
  <tr>
    <td align="center">랜딩 화면</td>
    <td align="center">Pull Request 검토</td>
  </tr>
</table>

## 데모

<table>
  <tr>
    <td align="center" width="33%">
      <a href="https://youtu.be/f56hsj97zWA"><img src="https://img.youtube.com/vi/f56hsj97zWA/0.jpg" alt="Maintainer creating a repository" /></a><br />
      <strong>Maintainer creates a repository</strong>
    </td>
    <td align="center" width="33%">
      <a href="https://youtu.be/HiM1U8WBm3g"><img src="https://img.youtube.com/vi/HiM1U8WBm3g/0.jpg" alt="Contributor developing" /></a><br />
      <strong>Contributor develops a change</strong>
    </td>
    <td align="center" width="33%">
      <a href="https://youtu.be/qIJjybwHhjo"><img src="https://img.youtube.com/vi/qIJjybwHhjo/0.jpg" alt="Maintainer approving a pull request" /></a><br />
      <strong>Maintainer approves a pull request</strong>
    </td>
  </tr>
</table>

## 설치

**사전 조건**

- Go 1.25 언어 버전을 사용하며, `go.mod`의 `toolchain` 설정이 Go 1.26.6을 선택합니다.
- Foundry 1.7.1+ (`anvil`, `forge`) — 로컬 체인 및 컨트랙트 배포용 (선택)
- Node.js 20.19+ — 컨트랙트 ABI 아티팩트 생성용 (선택)
- IPFS 데몬 (Kubo) — `ipfs daemon`
- 이더리움 노드 접근 (로컬: Anvil, 테스트넷: Sepolia 등)

```bash
git clone https://github.com/opendasom/bit.git
cd bit
go build -o bit ./cmd/bit

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
# Web3 저장소에서 실행
git clone --recurse-submodules https://github.com/opendasom/bit-w3.git
cd bit-w3
npm ci
npm run anvil:seed
```

시드 명령은 기본 Anvil 계정 4개를 사용하며 비어 있는 registry에서 한 번만 실행합니다.
README에 나온 private key는 Anvil의 공개된 **로컬 테스트 전용** 키입니다. 어떤 테스트넷이나 메인넷에서도 사용하면 안 됩니다.

---

## 빠른 시작

### 1. 저장소 초기화

```bash
mkdir my-project && cd my-project
git init -b main

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

> **주의**: indexed PR range와 atomic fork가 추가되어 ABI가 변경되었습니다. 기존 배포와 호환되지 않습니다. PR 설명은 체인에 저장됩니다. 새 BitRegistry를 배포했다면 `bit-w3`의 `.env.local`에서 `VITE_BIT_CONTRACT`를 새 주소로 변경하고 웹 개발 서버를 다시 시작하세요.

### 6. 웹 explorer

웹 explorer는 독립된 [`opendasom/bit-w3`](https://github.com/opendasom/bit-w3) 저장소에서 관리합니다. 해당 저장소는 이 저장소를 `bit` submodule로 포함하여 CLI와 동일한 `BitRegistry` ABI를 사용합니다.

```bash
git clone --recurse-submodules https://github.com/opendasom/bit-w3.git
cd bit-w3
npm ci
cp .env.example .env.local
npm run dev
```

웹 관련 문서, 이슈, 보안 신고 및 기타 기여는 모두 이 `bit` 저장소에서 통합 관리합니다.

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
npm ci
npm run compile
```

---

## 프로젝트 구조

```
bit/
├── cmd/
│   └── bit/
│       └── main.go   # bit CLI 진입점
├── internal/
│   ├── cli/          # Cobra 커맨드와 CLI 입출력
│   ├── app/          # 명령 실행 로직
│   ├── chain/        # BitRegistry 컨트랙트 연동 (go-ethereum)
│   ├── git/          # .git 읽기/쓰기 (go-git + exec git)
│   ├── ipfs/         # IPFS HTTP API 클라이언트
│   ├── cid/          # CIDv0 ↔ bytes32 변환 (외부 의존성 없음)
│   ├── manifest/     # manifest JSON 인코딩/디코딩
│   └── config/       # .bit/config.json 관리
├── contracts/
│   ├── src/BitRegistry.sol   # Solidity 컨트랙트
│   ├── tests/                # Foundry 컨트랙트 테스트
│   └── scripts/              # CLI 호환 ABI 아티팩트 생성
├── tests/
│   └── e2e/cli/              # CLI 종단 간 테스트
└── package.json              # CLI 호환 ABI 생성용 Node.js 패키지
```

---

## 주요 의존성

| 패키지 | 용도 | 라이선스 |
|--------|------|----------|
| `go-ethereum v1.17.0` | 이더리움 클라이언트, ABI 바인딩 | LGPL-3.0 (library) |
| `go-git/v5 v5.19.1` | Git 저장소 읽기 | Apache-2.0 |
| `cobra v1.8.1` | CLI 프레임워크 | Apache-2.0 |
| `golang.org/x/crypto v0.51.0` | 암호화 유틸리티 | BSD-3-Clause |

---

## 개발과 검증

```bash
go test ./...
go vet ./...

cd contracts && forge test && cd ..

npm ci
npm run compile
```

기여 방법은 [CONTRIBUTING.md](CONTRIBUTING.md), 취약점 신고는 [SECURITY.md](SECURITY.md)를 참고하세요.

---

## License

이 저장소에서 작성한 Bit 소스 코드는 [MIT License](LICENSE)로 배포됩니다. 제3자 컴포넌트는 각자의 라이선스를 유지하며, 특히 `go-ethereum`을 링크한 CLI 바이너리를 배포할 때는 LGPL-3.0 의무를 함께 충족해야 합니다.
