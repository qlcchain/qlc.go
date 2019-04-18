package module

import (
	"github.com/qlcchain/go-qlc/common/types"
	"github.com/qlcchain/go-qlc/rpc"
	"github.com/qlcchain/go-qlc/rpc/api"
)

type PledgeApi struct {
	client *rpc.Client
}

// NewPledgeApi creates pledge module for client
func NewPledgeApi(c *rpc.Client) *PledgeApi {
	return &PledgeApi{client: c}
}

// GetMintageData returns pledge data by pledge parameters
func (p *PledgeApi) GetPledgeData(param *api.PledgeParam) ([]byte, error) {
	var r []byte
	err := p.client.Call(&r, "pledge_getPledgeData", param)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// GetPledgeBlock returns pledge block by pledge parameters
func (p *PledgeApi) GetPledgeBlock(param *api.PledgeParam) (*types.StateBlock, error) {
	var sb types.StateBlock
	err := p.client.Call(&sb, "pledge_getPledgeBlock", param)
	if err != nil {
		return nil, err
	}
	return &sb, nil
}

// GetPledgeRewordBlock returns pledge reward block by pledge block
func (p *PledgeApi) GetPledgeRewordBlock(input *types.StateBlock) (*types.StateBlock, error) {
	var sb types.StateBlock
	err := p.client.Call(&sb, "pledge_getPledgeRewordBlock", input)
	if err != nil {
		return nil, err
	}
	return &sb, nil
}

// GetWithdrawPledgeData returns withdraw pledge data by withdraw parameters
func (p *PledgeApi) GetWithdrawPledgeData(param *api.WithdrawPledgeParam) ([]byte, error) {
	var r []byte
	err := p.client.Call(&r, "pledge_getWithdrawPledgeData", param)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// GetWithdrawPledgeBlock returns withdraw pledge block by withdraw parameters
func (p *PledgeApi) GetWithdrawPledgeBlock(param *api.WithdrawPledgeParam) (*types.StateBlock, error) {
	var sb types.StateBlock
	err := p.client.Call(&sb, "pledge_getWithdrawPledgeBlock", param)
	if err != nil {
		return nil, err
	}
	return &sb, nil
}

// GetWithdrawRewardBlock returns withdraw reward block by pledge block
func (p *PledgeApi) GetWithdrawRewardBlock(input *types.StateBlock) (*types.StateBlock, error) {
	var sb types.StateBlock
	err := p.client.Call(&sb, "pledge_getWithdrawRewardBlock", input)
	if err != nil {
		return nil, err
	}
	return &sb, nil
}