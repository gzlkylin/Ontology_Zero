/*
 * Copyright (C) 2018 Onchain <onchain@onchain.com>
 *
 * This file is part of The ontology_Zero.
 *
 * The ontology_Zero is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology_Zero is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology_Zero.  If not, see <http://www.gnu.org/licenses/>.
 */

package common

import (
	"bytes"
	"fmt"
	. "github.com/Ontology/common"
	"github.com/Ontology/core/ledger"
	tx "github.com/Ontology/core/transaction"
	. "github.com/Ontology/errors"
	. "github.com/Ontology/net/httpjsonrpc"
	Err "github.com/Ontology/net/httprestful/error"
	. "github.com/Ontology/net/protocol"
	"math"
	"strconv"
)

var node Noder

const TlsPort int = 443

type ApiServer interface {
	Start() error
	Stop()
}

func SetNode(n Noder) {
	node = n
}

//Node
func GetConnectionCount(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	if node != nil {
		resp["Result"] = node.GetConnectionCnt()
	}

	return resp
}

//Block
func GetBlockHeight(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	resp["Result"] = ledger.DefaultLedger.Blockchain.BlockHeight
	return resp
}
func GetBlockHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	param := cmd["Height"].(string)
	if len(param) == 0 {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	height, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	hash, err := ledger.DefaultLedger.Store.GetBlockHash(uint32(height))
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	resp["Result"] = ToHexString(hash.ToArray())
	return resp
}
func GetTotalIssued(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	assetid, ok := cmd["Assetid"].(string)
	if !ok {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var assetHash Uint256

	bys, err := HexToBytes(assetid)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	if err := assetHash.Deserialize(bytes.NewReader(bys)); err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	amount, err := ledger.DefaultLedger.Store.GetQuantityIssued(assetHash)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	val := float64(amount) / math.Pow(10, 8)
	//valStr := strconv.FormatFloat(val, 'f', -1, 64)
	resp["Result"] = val
	return resp
}
func GetBlockInfo(block *ledger.Block) BlockInfo {
	hash := block.Hash()
	blockHead := &BlockHead{
		Version:          block.Header.Version,
		PrevBlockHash:    ToHexString(block.Header.PrevBlockHash.ToArray()),
		TransactionsRoot: ToHexString(block.Header.TransactionsRoot.ToArray()),
		BlockRoot:        ToHexString(block.Header.BlockRoot.ToArray()),
		StateRoot:        ToHexString(block.Header.StateRoot.ToArray()),
		Timestamp:        block.Header.Timestamp,
		Height:           block.Header.Height,
		ConsensusData:    block.Header.ConsensusData,
		NextBookKeeper:   ToHexString(block.Header.NextBookKeeper.ToArray()),
		Program: ProgramInfo{
			Code:      ToHexString(block.Header.Program.Code),
			Parameter: ToHexString(block.Header.Program.Parameter),
		},
		Hash: ToHexString(hash.ToArray()),
	}

	trans := make([]*Transactions, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		trans[i] = TransArryByteToHexString(block.Transactions[i])
	}

	b := BlockInfo{
		Hash:         ToHexString(hash.ToArray()),
		BlockData:    blockHead,
		Transactions: trans,
	}
	return b
}

func GetBlockTransactions(block *ledger.Block) interface{} {
	trans := make([]string, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		h := block.Transactions[i].Hash()
		trans[i] = ToHexString(h.ToArray())
	}
	hash := block.Hash()
	type BlockTransactions struct {
		Hash         string
		Height       uint32
		Transactions []string
	}
	b := BlockTransactions{
		Hash:         ToHexString(hash.ToArray()),
		Height:       block.Header.Height,
		Transactions: trans,
	}
	return b
}
func getBlock(hash Uint256, getTxBytes bool) (interface{}, int64) {
	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return "", Err.UNKNOWN_BLOCK
	}
	if getTxBytes {
		w := bytes.NewBuffer(nil)
		block.Serialize(w)
		return ToHexString(w.Bytes()), Err.SUCCESS
	}
	return GetBlockInfo(block), Err.SUCCESS
}
func GetBlockByHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	param := cmd["Hash"].(string)
	if len(param) == 0 {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var getTxBytes bool = false
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		getTxBytes = true
	}
	var hash Uint256
	hex, err := HexToBytes(param)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	if err := hash.Deserialize(bytes.NewReader(hex)); err != nil {
		resp["Error"] = Err.INVALID_TRANSACTION
		return resp
	}

	resp["Result"], resp["Error"] = getBlock(hash, getTxBytes)

	return resp
}
func GetBlockTxsByHeight(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)

	param := cmd["Height"].(string)
	if len(param) == 0 {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	height, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	index := uint32(height)
	hash, err := ledger.DefaultLedger.Store.GetBlockHash(index)
	if err != nil {
		resp["Error"] = Err.UNKNOWN_BLOCK
		return resp
	}
	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		resp["Error"] = Err.UNKNOWN_BLOCK
		return resp
	}
	resp["Result"] = GetBlockTransactions(block)
	return resp
}
func GetBlockByHeight(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)

	param := cmd["Height"].(string)
	if len(param) == 0 {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var getTxBytes bool = false
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		getTxBytes = true
	}
	height, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	index := uint32(height)
	hash, err := ledger.DefaultLedger.Store.GetBlockHash(index)
	if err != nil {
		resp["Error"] = Err.UNKNOWN_BLOCK
		return resp
	}
	resp["Result"], resp["Error"] = getBlock(hash, getTxBytes)
	return resp
}

//Asset
func GetAssetByHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)

	str := cmd["Hash"].(string)
	hex, err := HexToBytes(str)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var hash Uint256
	err = hash.Deserialize(bytes.NewReader(hex))
	if err != nil {
		resp["Error"] = Err.INVALID_ASSET
		return resp
	}
	asset, err := ledger.DefaultLedger.Store.GetAsset(hash)
	if err != nil {
		resp["Error"] = Err.UNKNOWN_ASSET
		return resp
	}
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		w := bytes.NewBuffer(nil)
		asset.Serialize(w)
		resp["Result"] = ToHexString(w.Bytes())
		return resp
	}
	resp["Result"] = asset
	return resp
}

func GetBalanceByAddr(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	addr, ok := cmd["Addr"].(string)
	if !ok {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var programHash Uint160
	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	account, err := ledger.DefaultLedger.Store.GetAccount(programHash)
	if err != nil {
		resp["Error"] = Err.UNKNOWN_PROGRAM
		return resp
	}
	type Result struct {
		AssetId string
		Value   string
	}
	var results []Result
	for k, v := range account.Balances {
		assetid := ToHexString(k.ToArray())
		results = append(results, Result{assetid, strconv.FormatInt(v.GetData(), 10)})
	}
	//valStr := strconv.FormatFloat(val, 'f', -1, 64)
	resp["Result"] = results
	return resp
}

func GetBalanceByAsset(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	addr, ok := cmd["Addr"].(string)
	assetid, k := cmd["Assetid"].(string)
	if !ok || !k {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var programHash Uint160
	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	account, err := ledger.DefaultLedger.Store.GetAccount(programHash)
	if err != nil {
		resp["Error"] = Err.UNKNOWN_PROGRAM
		return resp
	}
	ass, _ := HexToBytes(assetid)
	assid, _ := Uint256ParseFromBytes(ass)
	if v, ok := account.Balances[assid]; ok {
		resp["Result"] = v.GetData()
	}
	//valStr := strconv.FormatFloat(val, 'f', -1, 64)
	return resp
}

func GetUnspendOutput(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	addr, ok := cmd["Addr"].(string)
	assetid, k := cmd["Assetid"].(string)
	if !ok || !k {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}

	var programHash Uint160
	var assetHash Uint256
	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	bys, err := HexToBytes(assetid)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	if err := assetHash.Deserialize(bytes.NewReader(bys)); err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	type UTXOUnspentInfo struct {
		Txid  string
		Index uint32
		Value string
	}
	infos, err := ledger.DefaultLedger.Store.GetUnspentFromProgramHash(programHash, assetHash)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		resp["Result"] = err
		return resp
	}
	var UTXOoutputs []UTXOUnspentInfo
	for _, v := range infos {
		val := strconv.FormatInt(int64(v.Value), 10)
		UTXOoutputs = append(UTXOoutputs, UTXOUnspentInfo{Txid: ToHexString(v.Txid.ToArray()), Index: v.Index, Value: val})
	}
	resp["Result"] = UTXOoutputs
	return resp
}

//Transaction
func GetTransactionByHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)

	str := cmd["Hash"].(string)
	bys, err := HexToBytes(str)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var hash Uint256
	err = hash.Deserialize(bytes.NewReader(bys))
	if err != nil {
		resp["Error"] = Err.INVALID_TRANSACTION
		return resp
	}
	tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
	if err != nil {
		resp["Error"] = Err.UNKNOWN_TRANSACTION
		return resp
	}
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		w := bytes.NewBuffer(nil)
		tx.Serialize(w)
		resp["Result"] = ToHexString(w.Bytes())
		return resp
	}
	tran := TransArryByteToHexString(tx)
	resp["Result"] = tran
	return resp
}
func SendRawTransaction(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)

	str, ok := cmd["Data"].(string)
	if !ok {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	bys, err := HexToBytes(str)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var txn tx.Transaction
	if err := txn.Deserialize(bytes.NewReader(bys)); err != nil {
		resp["Error"] = Err.INVALID_TRANSACTION
		return resp
	}
	var hash Uint256
	hash = txn.Hash()
	if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
		resp["Error"] = int64(errCode)
		return resp
	}
	resp["Result"] = ToHexString(hash.ToArray())
	//TODO 0xd1 -> tx.InvokeCode
	if txn.TxType == 0xd1 {
		if userid, ok := cmd["Userid"].(string); ok && len(userid) > 0 {
			resp["Userid"] = userid
		}
	}
	return resp
}

//stateupdate
func GetStateUpdate(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	namespace, ok := cmd["Namespace"].(string)
	if !ok {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	key, ok := cmd["Key"].(string)
	if !ok {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	fmt.Println(cmd, namespace, key)
	//TODO get state from store
	return resp
}

func GetSmartCodeEvent(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)

	param := cmd["Height"].(string)
	if len(param) == 0 {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	height, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	index := uint32(height)
	resp["Result"] = map[string]interface{}{"Height": index}
	//TODO resp
	return resp
}

func ResponsePack(errCode int64) map[string]interface{} {
	resp := map[string]interface{}{
		"Action":  "",
		"Result":  "",
		"Error":   errCode,
		"Desc":    "",
		"Version": "1.0.0",
	}
	return resp
}
func GetContract(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Err.SUCCESS)
	str := cmd["Hash"].(string)
	bys, err := HexToBytes(str)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var hash Uint160
	err = hash.Deserialize(bytes.NewReader(bys))
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	//TODO GetContract from store
	//contract, err := ledger.DefaultLedger.Store.GetContract(hash)
	//if err != nil {
	//	resp["Error"] = Err.INVALID_PARAMS
	//	return resp
	//}
	//resp["Result"] = string(contract)
	return resp
}
