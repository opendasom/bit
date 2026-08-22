# bit CLI Test Suite

`bit init` / `remote add` / `push` / `pull` 전체를 실제 anvil 체인 + 로컬 IPFS
데몬에 대고 end-to-end로 검증하는 bash 테스트 스위트. 각 케이스는 실제로
한 번씩 라이브 환경에서 실행해 동작을 확인한 뒤 작성됨 (짐작이 아니라
검증된 동작 기준).

## 사전 준비

먼저 아래 두 개를 각각 띄워둬야 함 (스크립트가 대신 실행해주지 않음):

```bash
anvil                # http://127.0.0.1:8545
ipfs daemon           # API http://127.0.0.1:5001
```

필요한 도구: `go`, `forge`, `cast`, `anvil`, `ipfs`, `jq`, `git`, `curl`
(없으면 스크립트가 시작 전에 바로 에러를 냄)

## 실행 방법

**`./tests/e2e/cli/run.sh` .** 컨트랙트 배포부터 `bit`
바이너리 빌드까지 스크립트가 실행할 때마다 알아서 새로 함 (매번 새
컨트랙트를 배포하므로 이전 실행 상태와 안 섞임).

```bash
./tests/e2e/cli/run.sh                 # 표준 케이스만 (48개 assertion)
RUN_SLOW=1 ./tests/e2e/cli/run.sh      # 페이지네이션 등 느린 케이스 포함
KEEP_WORKDIR=1 ./tests/e2e/cli/run.sh  # 끝나고 임시 작업 디렉토리를 안 지움 (실패 시 직접 들여다보기용)
```

끝나면 마지막에 `RESULTS: N passed, N failed, N skipped` 요약이 찍힘.
`FAIL`이 하나라도 있으면 exit code가 1.

## 테스트 케이스 목록

### `bit init` (`cases/10_init.sh`)

| ID | 확인하는 것 |
|---|---|
| INIT-1 | `.git` 없는 디렉토리에서 `bit init`이 config를 만들지 않고 깔끔히 실패하는지 |
| INIT-2 | 같은 지갑으로 두 번 `bit init` 해도 repoId가 1씩 정확히 증가하는지 (전역 카운터 충돌 없는지) |

### `bit remote add` (`cases/20_remote.sh`)

| ID | 확인하는 것 |
|---|---|
| REMOTE-1 | 정상 URL(`bit://local/<addr>/<id>`)이 config.json에 올바르게 저장되는지 |
| REMOTE-2 | `#branch` fragment가 repoId 파싱에 안 섞이고 제대로 잘려나가는지 |
| REMOTE-3 | path segment가 부족한 URL이 panic 없이 에러로 막히는지 |
| REMOTE-4 | repoId 자리에 숫자가 아닌 값이 오면 깔끔히 에러로 막히는지 |

### `bit push` (`cases/30_push.sh`)

| ID | 확인하는 것 |
|---|---|
| PUSH-1 | 두 클론이 동시에 push할 때 CAS로 한쪽만 성공하고 진 쪽은 명확한 에러를 받는지 (레이스를 실제로 재현) |
| PUSH-2 | merge 커밋이 껴 있으면 push가 막히는지 — **단, merge 커밋 앞의 일반 커밋들은 이미 체인에 기록된 채로 중단됨** (원자적이지 않음, 재현해서 확인한 실제 동작) |
| PUSH-3 | 이미 push한 커밋을 로컬에서 `amend`한 뒤 재push하면, 체인에 트랜잭션을 보내지도 않고 client 단에서 먼저 막히는지 |
| PUSH-4 | Contributor 권한만 있는 지갑은 push가 거부되는지 (`MaintainerRequired`) |
| PUSH-6 | 새 커밋이 없을 때 push하면 트랜잭션을 아예 안 보내고 "up to date"로 끝나는지 |
| PUSH-8 | push 도중(여러 커밋 중 일부만 기록된 상태에서) 프로세스가 죽어도, 재실행하면 이어서 정상적으로 마무리되는지 |

### `bit pull` (`cases/40_pull.sh`)

| ID | 확인하는 것 |
|---|---|
| PULL-1 | 완전히 새 디렉토리에서 pull하면 원본과 100% 동일한 커밋 해시로 재구성되는지 |
| PULL-2 | 부분적으로 pull한 뒤 추가 push분만 이어서 pull되는지 |
| PULL-3 | 로컬에만 있는(원격에 없는) 커밋이 있을 때 pull이 그걸 덮어쓰지 않고 명확히 거부하는지 (가장 중요한 안전장치) |
| PULL-4 | 워킹트리에 uncommitted 변경이 있으면 pull이 거부되는지 |
| PULL-5 | `cast send`로 온체인에 실제 push 없이 조작된 커밋 레코드를 직접 심어서, pull의 3중 검증(commit hash/diff CID/tree hash)이 실제로 걸러내는지 |
| PULL-6 *(RUN_SLOW=1)* | 커밋 100개를 넘겨 push한 뒤 pull이 pageSize=100 페이지네이션 경계를 정확히 넘기는지 |
| PULL-7 | 커밋 해시 재현이 타임존(`GIT_AUTHOR_DATE`/`GIT_COMMITTER_DATE`)에 의존하는 부분 — 머신 2대가 필요해서 자동화하지 않음, 수동 확인 안내만 출력 |

## 참고

- 매 실행마다 새 컨트랙트를 배포하므로, anvil을 계속 띄워둔 채로 반복
  실행해도 이전 실행의 repoId와 안 겹침.
- `.bit/config.json`에 개인키가 평문으로 들어가지만, 전부
  `$WORKDIR`(임시 디렉토리) 안에서만 생기고 스크립트 종료 시 삭제됨
  (`KEEP_WORKDIR=1`이 아닌 한).
