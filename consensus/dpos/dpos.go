package dpos

import (
	"math/big"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"bytes"
	"errors"
	"sync"
	"github.com/ethereum/go-ethereum/accounts"
)

var (
	blockInterval = int64(10)
	epochInterval = int64(86400)
)

var (
	ErrInvalidTimestamp           = errors.New("invalid timestamp")
	ErrWaitForPrevBlock           = errors.New("wait for last block arrived")
	ErrMintFutureBlock            = errors.New("mint the future block")
	ErrMismatchSignerAndValidator = errors.New("mismatch block signer and validator")
	ErrInvalidBlockValidator      = errors.New("invalid block validator")
	ErrInvalidMintBlockTime       = errors.New("invalid time to mint the block")
	ErrNilBlockHeader             = errors.New("nil block header returned")
)

type SignerFn func(accounts.Account, []byte) ([]byte, error)
type Dpos struct {
	mu     sync.RWMutex
	signer common.Address
	signFn SignerFn
}

type DposContext struct {
	// 记账人列表
	validators []common.Address
}

func (d *Dpos) Author(header *types.Header) (common.Address, error) {

	return header.Coinbase, nil
}

func (d *Dpos) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	log.Info("VerifyHeader")
	return nil
}

func (d *Dpos) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	log.Info("VerifyHeaders")
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			err := d.verifyHeader(chain, header, headers[:i])

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

func (d *Dpos) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	log.Info("VerifyUncles")
	return nil
}

func (d *Dpos) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	log.Info("VerifySeal")
	return nil
}

// ParentHash, Number, GasLimit, Extra, Time, Coinbase set in "commitNewWork()"
func (d *Dpos) Prepare(chain consensus.ChainReader, header *types.Header) error {
	log.Info("Prepare()")
	//header.ParentHash = common.Hash{}
	header.UncleHash = common.Hash{}
	//header.Coinbase = common.Address{}
	header.Root = common.Hash{}
	header.TxHash = common.Hash{}
	header.ReceiptHash = common.Hash{}
	header.Bloom = types.Bloom{}
	header.Difficulty = big.NewInt(1)
	//header.Number = big.NewInt(1)
	//header.GasLimit = uint64(1000)
	//header.GasUsed = uint64(1)
	//header.Time = big.NewInt(time.Now().Unix())
	//header.Extra = append(header.Extra, make([]byte, 65)...)
	header.MixDigest = common.Hash{}
	copy(header.Nonce[:], hexutil.MustDecode("0x0000000000000000"))
	return nil
}

func (d *Dpos) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
	uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	log.Info("Finalize()")
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)

	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts), nil
}

func (d *Dpos) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	log.Info("Seal()", "Dpos Seal()", "Seal()")
	header := block.Header()
	return block.WithSeal(header), nil
}

func (d *Dpos) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	panic("implement me")
}

func (d *Dpos) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "clique",
		Version:   "1.0",
		Service:   &API{dpos: d},
		Public:    false,
	}}
}

func (d *Dpos) Close() error {
	return nil
}

func (d *Dpos) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	log.Info("__verifyHeader()")
	return nil
}

func PrevSlot(now int64) int64 {
	return int64((now-1)/blockInterval) * blockInterval
}

func NextSlot(now int64) int64 {
	return int64((now+blockInterval-1)/blockInterval) * blockInterval
}

func (d *Dpos) checkDeadline(lastBlock *types.Block, now int64) error {
	prevSlot := PrevSlot(now)
	nextSlot := NextSlot(now)
	if lastBlock.Time().Int64() >= nextSlot {
		return ErrMintFutureBlock
	}
	// last block was arrived, or time's up
	if lastBlock.Time().Int64() == prevSlot || nextSlot-now <= 1 {
		return nil
	}
	return ErrWaitForPrevBlock
}
func (d *Dpos) lookupValidator(now int64, validators []common.Address) (validator common.Address, err error) {
	validator = common.Address{}
	offset := now % epochInterval
	if offset%blockInterval != 0 {
		return common.Address{}, ErrInvalidMintBlockTime
	}
	offset /= blockInterval

	validatorSize := len(validators)
	if validatorSize == 0 {
		return common.Address{}, errors.New("failed to lookup validator")
	}
	offset %= int64(validatorSize)
	return validators[offset], nil
}

func (d *Dpos) CheckValidator(lastBlock *types.Block, now int64) error {
	// TODO
	return nil
	// TODO
	if err := d.checkDeadline(lastBlock, now); err != nil {
		return err
	}
	validators := getValidators(lastBlock)

	validator, err := d.lookupValidator(now, validators)
	if err != nil {
		return err
	}
	if (validator == common.Address{}) || bytes.Compare(validator.Bytes(), d.signer.Bytes()) != 0 {
		return ErrInvalidBlockValidator
	}
	return nil
}

func getValidators(lastBlock *types.Block) []common.Address {
	// TODO 先用写死的方式mock, 以后完善这个逻辑。 从上次区块中取
	validators := []common.Address{}
	add1 := common.Address{}
	add2 := common.Address{}
	validators = append(validators, add1)
	validators = append(validators, add2)
	return validators
	//dposContext := lastBlock.DposContext
	//validators := dposContext.validators
	//return validators
}

func (d *Dpos) Authorize(signer common.Address, signFn SignerFn) {
	d.mu.Lock()
	d.signer = signer
	d.signFn = signFn
	d.mu.Unlock()
}

func New() *Dpos {
	return &Dpos{}
}
