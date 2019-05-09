package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/qlcchain/go-qlc/common/types"
	"github.com/qlcchain/go-qlc/common/util"
	"github.com/qlcchain/go-qlc/crypto/random"
	"github.com/qlcchain/go-qlc/log"
	"github.com/qlcchain/go-qlc/rpc/api"
	"github.com/qlcchain/qlc-go-sdk"
	"github.com/qlcchain/qlc-go-sdk/example/robot/message"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	phonePrefix = []string{"130", "131", "132", "133", "134", "135", "136", "137", "138",
		"139", "147", "150", "151", "152", "153", "155", "156", "157", "158", "159", "186",
		"187", "188"}
	txInterval  = flag.Int("txInterval", 10, "send message interval")
	rxInterval  = flag.Int("rxInterval", 120, "receive message interval")
	endPoint    = ""
	accounts    arrayFlags
	minInterval = 10

	txAccountSize int
	maxAmount     = 8
	token         = "QLC"
	logger        = log.NewLogger("qlc_robot")
)

func main() {
	flag.StringVar(&endPoint, "endpoint", "ws://127.0.0.1:19736", "RPC Server endpoint")
	flag.Var(&accounts, "account", "account private key")
	flag.Parse()

	if *txInterval < minInterval {
		logger.Errorf("invalid txInterval %d[%d,∞]", *txInterval, minInterval)
		return
	}

	if *rxInterval < minInterval {
		logger.Errorf("invalid rxInterval %d[%d,∞]", *rxInterval, minInterval)
		return
	}

	if len(accounts) == 0 {
		logger.Error("can not find any account")
		return
	}
	var txAccounts []*types.Account
	for i, a := range accounts {
		bytes, e := hex.DecodeString(a)
		if e != nil {
			logger.Errorf("can not decode (%s) at %d to Account", a, i)
			continue
		}
		account := types.NewAccount(bytes)
		logger.Infof("Tx[%d]: %s", i, account.Address().String())
		txAccounts = append(txAccounts, account)
	}
	txAccountSize = len(txAccounts)

	if txAccountSize < 2 {
		logger.Errorf("not enough account(%d) to send Tx", txAccountSize)
		return
	}

	client, err := qlcchain.NewQLCClient(endPoint)
	if err != nil {
		logger.Error(err)
		return
	}

	// make sure all accounts already open
	accountPool := newAccountPool(txAccounts)
	err = generateReceives(client, accountPool)
	if err != nil {
		logger.Error(err)
		return
	}

	logger.Infof("%d Account will send Tx every %d second(s)", txAccountSize, *txInterval)

	txDuration, _ := time.ParseDuration(fmt.Sprintf("%ds", *txInterval))
	txTicker := time.NewTicker(txDuration)
	defer txTicker.Stop()

	rxDuration, _ := time.ParseDuration(fmt.Sprintf("%ds", *rxInterval))
	rxTicker := time.NewTicker(rxDuration)
	defer rxTicker.Stop()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	//prepare phone pool
	phonePool := newResourcePool(func() interface{} {
		return randomPhone()
	})

	for {
		select {
		case <-c:
			logger.Info("receive close signal, stop ...")
			//_ = generateReceives(client, accountPool)
			return
		case txTime := <-txTicker.C:
			logger.Info("produce send @ ", txTime)
			go func() {
				txAccount := accountPool.Get()
				rxAccount := accountPool.Get()
				sender := phonePool.Get()
				receiver := phonePool.Get()
				defer func() {
					accountPool.Put(txAccount)
					accountPool.Put(rxAccount)
					phonePool.Put(sender)
					phonePool.Put(receiver)
				}()

				amount := randomAmount()
				_, p := message.RandomPoem()
				m := p.Message()
				mh, err := client.SMS.MessageStore(m)
				if err != nil {
					logger.Error(err)
					return
				}

				param := &api.APISendBlockPara{
					From:      txAccount.Address(),
					TokenName: token,
					To:        rxAccount.Address(),
					Amount:    amount,
					Sender:    sender.(string),
					Receiver:  receiver.(string),
					Message:   mh,
				}
				logger.Debug(util.ToString(param))
				txBlock, err := client.Ledger.GenerateSendBlock(param, func(hash types.Hash) (types.Signature, error) {
					return txAccount.Sign(hash), nil
				})

				if err != nil {
					logger.Error(err)
				}
				logger.Info(util.ToString(txBlock))
				hash, err := client.Ledger.Process(txBlock)
				if err != nil {
					logger.Error(err)
				}
				logger.Info(hash.String())

			}()
		case rxTime := <-rxTicker.C:
			logger.Info("generate receive @ ", rxTime)
			go func() {
				err := generateReceives(client, accountPool)
				if err != nil {
					logger.Error(err)
				}
			}()
		}
	}
}

func generateReceives(client *qlcchain.QLCClient, pool *accountPool) error {
	//generate receive block
	cache := make(map[types.Address]*types.Account, 0)
	var addresses []types.Address

	pool.Iter(func(account *types.Account) error {
		addr := account.Address()
		addresses = append(addresses, addr)
		cache[addr] = account
		return nil
	})

	pendings, err := client.Ledger.AccountsPending(addresses, -1)
	if err != nil {
		return err
	}

	for addr, v := range pendings {
		for _, pending := range v {
			rxBlock, err := client.Ledger.GenerateReceiveBlockByHash(pending.Hash, func(hash types.Hash) (types.Signature, error) {
				if a, ok := cache[addr]; ok {
					return a.Sign(hash), nil
				} else {
					return types.Signature{}, fmt.Errorf("can not find addr[%s]private key", addr.String())
				}
			})
			if err != nil {
				logger.Error(err)
				continue
			}
			if h, err := client.Ledger.Process(rxBlock); err == nil {
				logger.Infof("generate receive %s from %s", pending.Hash.String(), h.String())
			}
		}
	}

	return nil
}

func randomAmount() types.Balance {
	i, _ := random.Intn(maxAmount)
	u, _ := util.SafeMul(uint64(i+1), uint64(1e7))
	b := new(big.Int).SetUint64(u)

	return types.Balance{Int: b}
}

func randomAccount(a *types.Account, account []*types.Account) *types.Account {
	i, _ := random.Intn(len(account))
	tmp := account[i]
	if a != nil && tmp.Address() == a.Address() {
		return randomAccount(a, account)
	}
	return tmp
}

func hash(msg string) types.Hash {
	m := fmt.Sprintf("%s powered by qlcchain", msg)
	h, _ := types.HashBytes([]byte(m))
	return h
}

func randomPhone() string {
	i, _ := random.Intn(len(phonePrefix))
	var sb strings.Builder
	sb.WriteString("+86")
	sb.WriteString(phonePrefix[i])
	for i := 0; i < 8; i++ {
		tmp, _ := random.Intn(10)
		sb.WriteString(strconv.Itoa(tmp))
	}

	return sb.String()
}

type resourcePool struct {
	pool sync.Pool
	size int64
}

func newResourcePool(fn func() interface{}) *resourcePool {
	return &resourcePool{pool: sync.Pool{New: fn}}
}

func (rp *resourcePool) Put(val interface{}) {
	rp.pool.Put(val)
	atomic.AddInt64(&(rp.size), 1)
}

func (rp *resourcePool) Get() interface{} {
	if rp.size > 0 {
		idx, _ := random.Intn(10)
		for i := 0; i < idx; i++ {
			val := rp.pool.Get()
			rp.Put(val)
		}
		atomic.AddInt64(&(rp.size), -1)
		//return rp.pool.Get()
	}
	return rp.pool.Get()
}

type accountPool struct {
	accounts []*types.Account
	locker   sync.RWMutex
}

func newAccountPool(accounts []*types.Account) *accountPool {
	return &accountPool{accounts: accounts}
}

func (ap *accountPool) Get() *types.Account {
	ap.locker.Lock()
	defer ap.locker.Unlock()
	i, _ := random.Intn(len(ap.accounts))
	tmp := ap.accounts[i]
	ap.accounts = append(ap.accounts[:i], ap.accounts[i+1:]...)
	return tmp
}

func (ap *accountPool) Put(account *types.Account) {
	ap.locker.Lock()
	defer ap.locker.Unlock()
	ap.accounts = append(ap.accounts, account)
}

func (ap *accountPool) Iter(fn func(account *types.Account) error) {
	ap.locker.RLock()
	defer ap.locker.RUnlock()
	for _, acc := range ap.accounts {
		e := fn(acc)
		if e != nil {
			logger.Error(e)
		}
	}
}
