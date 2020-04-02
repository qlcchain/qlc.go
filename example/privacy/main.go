package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"math/big"
	"strconv"

	qlcchain "github.com/qlcchain/qlc-go-sdk"
	"github.com/qlcchain/qlc-go-sdk/pkg/types"
	"github.com/qlcchain/qlc-go-sdk/pkg/util"
)

var flagNodeUrl string

func main() {
	flag.StringVar(&flagNodeUrl, "nodeurl", "http://127.0.0.1:19735", "RPC URL of node")

	flag.Parse()

	client, err := qlcchain.NewQLCClient(flagNodeUrl)
	if err != nil || client == nil {
		fmt.Println(err)
		return
	}

	fmt.Println(client.Version())

	demoContractAbi(client)

	demoMintageContract(client)

	demoMinerContract(client)
}

func demoContractAbi(client *qlcchain.QLCClient) {
	fmt.Println("============ demoContractAbi ============")
	rspAddrs := client.Contract.ContractAddressList()
	fmt.Printf("ContractAddressList:\n%s\n", util.ToIndentString(rspAddrs))

	for _, ca := range rspAddrs {
		rspAbi, err := client.Contract.GetAbiByContractAddress(ca)
		if err != nil {
			fmt.Println("GetAbiByContractAddress", err)
			return
		}
		fmt.Printf("GetAbiByContractAddress:\n%s, %s\n", ca, rspAbi)
	}
}

func demoMintageContract(client *qlcchain.QLCClient) {
	fmt.Println("============ demoMintageContract ============")
	/*
			{
		    "type":"function",
		    "name":"Mintage",
		    "inputs":[
		        {
		            "name":"tokenId",
		            "type":"tokenId"
		        },
		        {
		            "name":"tokenName",
		            "type":"string"
		        },
		        {
		            "name":"tokenSymbol",
		            "type":"string"
		        },
		        {
		            "name":"totalSupply",
		            "type":"uint256"
		        },
		        {
		            "name":"decimals",
		            "type":"uint8"
		        },
		        {
		            "name":"beneficial",
		            "type":"address"
		        },
		        {
		            "name":"NEP5TxId",
		            "type":"string"
		        }
		    ]
		}
	*/
	minerAddr, _ := types.HexToAddress("qlc_3hw8s1zubhxsykfsq5x7kh6eyibas9j3ga86ixd7pnqwes1cmt9mqqrngap4")
	minerSendPara := qlcchain.MintageParams{
		SelfAddr:    minerAddr,
		TokenName:   "QBTC",
		TokenSymbol: "QBTC",
		TotalSupply: "2100000000000000",
		Decimals:    8,
		Beneficial:  minerAddr,
		NEP5TxId:    util.RandomFixedString(32),
	}
	mintRspData, err := client.Mintage.GetMintageData(&minerSendPara)
	if err != nil {
		fmt.Println("GetMintageData", err)
		return
	}
	fmt.Printf("GetMintageData:\n%s\n", hex.EncodeToString(mintRspData))

	mintRspBlk, err := client.Mintage.GetMintageBlock(&minerSendPara)
	if err != nil {
		fmt.Println("GetMintageBlock", err)
		return
	}
	fmt.Printf("GetMintageBlock:\n%s\n", util.ToIndentString(mintRspBlk))

	contractSendPara := qlcchain.ContractSendBlockPara{
		Address:   minerSendPara.SelfAddr,
		To:        types.MintageAddress,
		TokenName: "QLC",
		Amount:    types.NewBalance(5 * 1e13),
		Data:      mintRspData,
	}
	contractRspBlk, err := client.Contract.GenerateSendBlock(&contractSendPara)
	if err != nil {
		fmt.Println("GenerateSendBlock", err)
		return
	}
	fmt.Printf("GenerateSendBlock:\n%s\n", util.ToIndentString(contractRspBlk))
}

func demoMinerContract(client *qlcchain.QLCClient) {
	fmt.Println("============ demoMinerContract ============")
	/*
		{"type":"function","name":"MinerReward","inputs":[
				{"name":"coinbase","type":"address"},
				{"name":"beneficial","type":"address"},
				{"name":"startHeight","type":"uint64"},
				{"name":"endHeight","type":"uint64"},
				{"name":"rewardBlocks","type":"uint64"},
				{"name":"rewardAmount","type":"uint256"}
			]}
	*/
	minerAddr, _ := types.HexToAddress("qlc_3hw8s1zubhxsykfsq5x7kh6eyibas9j3ga86ixd7pnqwes1cmt9mqqrngap4")
	minerSendPara := qlcchain.RewardParam{
		Coinbase:     minerAddr,
		Beneficial:   minerAddr,
		StartHeight:  1440,
		EndHeight:    2879,
		RewardBlocks: 100,
		RewardAmount: big.NewInt(30000000000),
	}
	minerRspData, err := client.Miner.GetRewardData(&minerSendPara)
	if err != nil {
		fmt.Println("GetRewardData", err)
		return
	}
	fmt.Printf("GetRewardData:\n%s\n", hex.EncodeToString(minerRspData))

	paraStrList := []string{
		minerSendPara.Coinbase.String(),
		minerSendPara.Beneficial.String(),
		strconv.Itoa(int(minerSendPara.StartHeight)),
		strconv.Itoa(int(minerSendPara.EndHeight)),
		strconv.Itoa(int(minerSendPara.RewardBlocks)),
		minerSendPara.RewardAmount.String(),
	}
	contractRspData, err := client.Contract.PackChainContractData(types.MinerAddress, "MinerReward", paraStrList)
	if err != nil {
		fmt.Println("PackChainContractData", err)
		return
	}
	fmt.Printf("PackChainContractData:\n%s\n", hex.EncodeToString(contractRspData))

	if !bytes.Equal(minerRspData, contractRspData) {
		fmt.Println("minerRspData != contractRspData")
		return
	}

	minerRspBlk, err := client.Miner.GetRewardSendBlock(&minerSendPara)
	if err != nil {
		fmt.Println("GetRewardSendBlock", err)
		return
	}
	fmt.Printf("GetRewardSendBlock:\n%s\n", util.ToIndentString(minerRspBlk))

	contractSendPara := qlcchain.ContractSendBlockPara{
		Address:   minerSendPara.Coinbase,
		To:        types.MinerAddress,
		TokenName: "QLC",
		Amount:    types.NewBalance(0),
		Data:      contractRspData,
	}
	contractRspBlk, err := client.Contract.GenerateSendBlock(&contractSendPara)
	if err != nil {
		fmt.Println("GenerateSendBlock", err)
		return
	}
	fmt.Printf("GenerateSendBlock:\n%s\n", util.ToIndentString(contractRspBlk))
}
