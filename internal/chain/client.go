package chain

import (
	"context"
	"crypto/ecdsa"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/crypto/sha3"
)

//go:embed artifacts/BitRegistry.json
var bitRegistryArtifact []byte

var (
	artifactOnce sync.Once
	artifactABI  abi.ABI
	artifactErr  error
)

type artifactJSON struct {
	ABI json.RawMessage `json:"abi"`
}

type CommitRecord struct {
	TreeHash       [20]byte
	ManifestDigest [32]byte
	DiffDigest     [32]byte
	Updater        common.Address
	Timestamp      *big.Int
}

type BranchCommitRecord struct {
	CommitHash     [20]byte
	TreeHash       [20]byte
	ManifestDigest [32]byte
	DiffDigest     [32]byte
}

type RepoRecord struct {
	Owner       common.Address
	MetadataCID string
}

// Client는 BitRegistry 스마트 컨트랙트와 통신하는 클라이언트다.
type Client struct {
	contractAddress common.Address
	contract        *bind.BoundContract
	auth            *bind.TransactOpts
	conn            *ethclient.Client
}

func loadArtifactABI() (abi.ABI, error) {
	artifactOnce.Do(func() {
		var artifact artifactJSON
		if err := json.Unmarshal(bitRegistryArtifact, &artifact); err != nil {
			artifactErr = err
			return
		}
		artifactABI, artifactErr = abi.JSON(strings.NewReader(string(artifact.ABI)))
	})
	return artifactABI, artifactErr
}

// NewClient는 RPC 노드에 연결하고 개인키로 지갑을 초기화해 Client를 반환한다.
func NewClient(rpcURL, contractAddress, privateKeyHex string) (*Client, error) {
	conn, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}

	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	privKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, err
	}

	chainID, err := conn.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
	if err != nil {
		return nil, err
	}

	if !common.IsHexAddress(contractAddress) {
		return nil, fmt.Errorf("invalid contract address: %s", contractAddress)
	}
	parsedABI, err := loadArtifactABI()
	if err != nil {
		return nil, err
	}
	address := common.HexToAddress(contractAddress)
	contract := bind.NewBoundContract(address, parsedABI, conn, conn, conn)

	return &Client{contractAddress: address, contract: contract, auth: auth, conn: conn}, nil
}

// branchNameToBytes32는 브랜치명 문자열을 keccak256 해시한 [32]byte로 변환한다.
func branchNameToBytes32(name string) [32]byte {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(name))
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

func GitHashToBytes20(hash string) ([20]byte, error) {
	var out [20]byte
	hash = strings.TrimPrefix(hash, "0x")
	decoded, err := hex.DecodeString(hash)
	if err != nil {
		return out, err
	}
	if len(decoded) != 20 {
		return out, fmt.Errorf("expected SHA-1 git hash to decode to 20 bytes, got %d", len(decoded))
	}
	copy(out[:], decoded)
	return out, nil
}

func Bytes20ToGitHash(hash [20]byte) string {
	return hex.EncodeToString(hash[:])
}

func IsZeroBytes20(hash [20]byte) bool {
	return hash == [20]byte{}
}

func (c *Client) call(method string, params ...interface{}) ([]interface{}, error) {
	var out []interface{}
	if err := c.contract.Call(&bind.CallOpts{Context: context.Background()}, &out, method, params...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) waitSuccessful(tx *types.Transaction) (*types.Receipt, error) {
	receipt, err := bind.WaitMined(context.Background(), c.conn, tx)
	if err != nil {
		return nil, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("transaction reverted: %s", tx.Hash().Hex())
	}
	return receipt, nil
}

// GetBranchHead는 특정 브랜치의 현재 헤드 manifest CID를 체인에서 조회한다.
func (c *Client) GetBranchHead(repoID *big.Int, branch string) (string, error) {
	out, err := c.call("getBranchHead", repoID, branchNameToBytes32(branch))
	if err != nil {
		return "", err
	}
	cid := *abi.ConvertType(out[0], new([]byte)).(*[]byte)
	return string(cid), nil
}

func (c *Client) GetRepoCount() (*big.Int, error) {
	out, err := c.call("getRepoCount")
	if err != nil {
		return nil, err
	}
	return *abi.ConvertType(out[0], new(*big.Int)).(**big.Int), nil
}

func (c *Client) GetRepo(repoID *big.Int) (*RepoRecord, error) {
	out, err := c.call("getRepo", repoID)
	if err != nil {
		return nil, err
	}
	metadataCID := *abi.ConvertType(out[1], new([]byte)).(*[]byte)
	return &RepoRecord{
		Owner:       *abi.ConvertType(out[0], new(common.Address)).(*common.Address),
		MetadataCID: string(metadataCID),
	}, nil
}

func (c *Client) GetBranchCommit(repoID *big.Int, branch string) ([20]byte, error) {
	out, err := c.call("getBranchCommit", repoID, branchNameToBytes32(branch))
	if err != nil {
		return [20]byte{}, err
	}
	return *abi.ConvertType(out[0], new([20]byte)).(*[20]byte), nil
}

func (c *Client) GetBranchHistoryLength(repoID *big.Int, branch string) (*big.Int, error) {
	out, err := c.call("getBranchHistoryLength", repoID, branchNameToBytes32(branch))
	if err != nil {
		return nil, err
	}
	return *abi.ConvertType(out[0], new(*big.Int)).(**big.Int), nil
}

func (c *Client) GetBranchCommitAt(repoID *big.Int, branch string, index *big.Int) ([20]byte, error) {
	out, err := c.call("getBranchCommitAt", repoID, branchNameToBytes32(branch), index)
	if err != nil {
		return [20]byte{}, err
	}
	return *abi.ConvertType(out[0], new([20]byte)).(*[20]byte), nil
}

func (c *Client) GetBranchCommitsWithMetadata(repoID *big.Int, branch string, start, limit *big.Int) ([]BranchCommitRecord, error) {
	out, err := c.call("getBranchCommitsWithMetadata", repoID, branchNameToBytes32(branch), start, limit)
	if err != nil {
		return nil, err
	}
	commitHashes := *abi.ConvertType(out[0], new([][20]byte)).(*[][20]byte)
	treeHashes := *abi.ConvertType(out[1], new([][20]byte)).(*[][20]byte)
	manifestDigests := *abi.ConvertType(out[2], new([][32]byte)).(*[][32]byte)
	diffDigests := *abi.ConvertType(out[3], new([][32]byte)).(*[][32]byte)
	records := make([]BranchCommitRecord, len(commitHashes))
	for i := range commitHashes {
		records[i] = BranchCommitRecord{
			CommitHash:     commitHashes[i],
			TreeHash:       treeHashes[i],
			ManifestDigest: manifestDigests[i],
			DiffDigest:     diffDigests[i],
		}
	}
	return records, nil
}

func (c *Client) GetCommit(repoID *big.Int, commitHash [20]byte) (*CommitRecord, error) {
	out, err := c.call("getCommit", repoID, commitHash)
	if err != nil {
		return nil, err
	}
	return &CommitRecord{
		TreeHash:       *abi.ConvertType(out[0], new([20]byte)).(*[20]byte),
		ManifestDigest: *abi.ConvertType(out[1], new([32]byte)).(*[32]byte),
		DiffDigest:     *abi.ConvertType(out[2], new([32]byte)).(*[32]byte),
		Updater:        *abi.ConvertType(out[3], new(common.Address)).(*common.Address),
		Timestamp:      *abi.ConvertType(out[4], new(*big.Int)).(**big.Int),
	}, nil
}

func (c *Client) RecordCommit(
	repoID *big.Int,
	branch string,
	expectedOldCommit [20]byte,
	commitHash [20]byte,
	treeHash [20]byte,
	parentHashes [][20]byte,
	manifestDigest [32]byte,
	diffDigest [32]byte,
) error {
	tx, err := c.contract.Transact(
		c.auth,
		"recordCommit",
		repoID,
		branchNameToBytes32(branch),
		expectedOldCommit,
		commitHash,
		treeHash,
		parentHashes,
		manifestDigest,
		diffDigest,
	)
	if err != nil {
		return err
	}
	_, err = c.waitSuccessful(tx)
	return err
}

// CreateRepo는 체인에 새 저장소를 생성하고 repoId를 반환한다.
func (c *Client) CreateRepo(metadataCID string) (*big.Int, error) {
	tx, err := c.contract.Transact(c.auth, "createRepo", []byte(metadataCID))
	if err != nil {
		return nil, err
	}
	receipt, err := c.waitSuccessful(tx)
	if err != nil {
		return nil, err
	}

	parsedABI, err := loadArtifactABI()
	if err != nil {
		return nil, err
	}
	event, ok := parsedABI.Events["RepoCreated"]
	if !ok {
		return nil, errors.New("RepoCreated event missing from ABI")
	}
	for _, log := range receipt.Logs {
		if log.Address == c.contractAddress && len(log.Topics) > 1 && log.Topics[0] == event.ID {
			return new(big.Int).SetBytes(log.Topics[1].Bytes()), nil
		}
	}

	return nil, errors.New("RepoCreated event not found in transaction receipt")
}

// GetPublicAddress는 개인키로부터 지갑 주소를 계산해 반환한다.
func GetPublicAddress(privateKeyHex string) (string, error) {
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	privKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", err
	}
	pubKey := privKey.Public().(*ecdsa.PublicKey)
	return crypto.PubkeyToAddress(*pubKey).Hex(), nil
}
