package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	//"math/rand"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
)

type Transaction struct {
	Key             string `json:"key"`
	Data            value  `json:"data"`
	Valid           bool   `json:"valid"`
	TransactionHash string `json:"transactionHash"`
}

type value struct {
	Val     float64 `json:"val"`
	Version float64 `json:"version"`
}

type BlockStatus string

const (
	Commited BlockStatus = "commited"
	Pending  BlockStatus = "pending"
)

type Block struct {
	BlockNumber   int           `json:"blockNumber"`
	Txns          []Transaction `json:"transaction"`
	TimeStamp     int           `json:"timestamp"`
	BlockStatus   BlockStatus   `json:"blockstatus"`
	PrevBlockHash string        `json:"prevBlockHash"`
	BlockHash     string        `json:"blockHash"`
}
type blockInterface interface {
	push(Txns []Transaction, db *leveldb.DB)
	update(Status BlockStatus)
}

func calculateTransactionHash(txn *Transaction, wg *sync.WaitGroup) {

	defer wg.Done()
	temp := fmt.Sprintf("%v", txn)
	txnBytes := []byte(temp)
	hash := sha256.Sum256(txnBytes)
	txnHash := hex.EncodeToString(hash[:])
	//return txnHash
	txn.TransactionHash = txnHash
}

func addTransactionToBlock(Txns []Transaction, db *leveldb.DB, blockSize int, myChan chan<- Block) {
	totalTransactions := len(Txns)
	numBlocks := totalTransactions / blockSize
	if totalTransactions%numBlocks != 0 {
		numBlocks++
		fmt.Println(numBlocks)
	}
	for i := 0; i < numBlocks; i++ {
		start := i * blockSize
		end := (i + 1) * blockSize
		if end > totalTransactions {
			end = totalTransactions
		}
		block := Block{
			BlockNumber:   i + 1,
			TimeStamp:     12,
			BlockStatus:   Pending,
			PrevBlockHash: "ec223",
		}

		block.push(Txns[start:end], db)

		block.update(Commited)
		myChan <- block
	}
	close(myChan)
}
func (block *Block) update(Status BlockStatus) {
	block.BlockStatus = Status
}

func (block *Block) push(Txns []Transaction, db *leveldb.DB) {
	for i := 0; i < len(Txns); i++ {

		res := validate(Txns[i], db)
		if res {

			key := Txns[i].Key
			data := value{
				Val:     Txns[i].Data.Val,
				Version: Txns[i].Data.Version + 1,
			}

			txn := Transaction{
				Key:             key,
				Data:            data,
				TransactionHash: Txns[i].TransactionHash,
			}
			//transactionHash:Txns[i].TransactionHash

			txn.Valid = true
			txn.TransactionHash = Txns[i].TransactionHash
			block.Txns = append(block.Txns, txn)
			datas, err := json.Marshal(txn)
			if err != nil {
				log.Println("Error while encoding", err)
				continue
			}
			err = db.Put([]byte(txn.Key), datas, nil)
			if err != nil {
				log.Println("Error while storing transaction in db", err)
			}
		} else {

			data, err := db.Get([]byte(Txns[i].Key), nil)
			if err != nil {

				return
			}
			var temp Transaction
			err = json.Unmarshal(data, &temp)
			if err != nil {
				return
			}
			temp.Valid = res
			block.Txns = append(block.Txns, temp)

		}
	}

}

func validate(txn Transaction, db *leveldb.DB) bool {
	valu, err := db.Get([]byte(txn.Key), nil)
	if err != nil {
		return false
	}
	var data Transaction
	err = json.Unmarshal(valu, &data)
	if err != nil {
		return false
	}
	if data.Data.Version == txn.Data.Version {
		return true
	}
	return false
}
func addBlockToFile(filePath string, block Block) {

	if blockNumberExists(block.BlockNumber) {
		fmt.Println("Block already exists in the file")
		return
	}

	data, err := json.Marshal(block)
	if err != nil {
		fmt.Println("Error marshalling Block:", err)
		return
	}
	file, err := os.OpenFile("block.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Append the JSON data to the file
	_, err = file.WriteString(string(data) + "\n")
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

}
func GetAllBlocksFromFile(filename string) {

	file, err := os.Open("block.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	blocks := []Block{}
	for _, line := range strings.Split(string(contents), "\n") {
		if line == "" {
			continue // Skip empty lines
		}

		block := Block{}
		err := json.Unmarshal([]byte(line), &block)
		if err != nil {
			fmt.Println("Error unmarshalling block:", err)
			return
		}
		fmt.Println(block)

		blocks = append(blocks, block)
	}
	//fmt.Printf(blocks)

}

func findByBlockNumber(blockNumber int) {
	file, err := os.Open("block.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	blocks := []Block{}
	for _, line := range strings.Split(string(contents), "\n") {
		if line == "" {
			continue // Skip empty lines
		}

		block := Block{}
		err := json.Unmarshal([]byte(line), &block)
		if err != nil {
			fmt.Println("Error unmarshalling block:", err)
			return
		}

		blocks = append(blocks, block)
	}
	desiredBlockNumber := blockNumber // Replace with the desired block number

	var desiredBlock Block
	for _, block := range blocks {
		if block.BlockNumber == desiredBlockNumber {
			desiredBlock = block
			break
		}
	}

	if desiredBlock.BlockNumber == 0 {
		fmt.Println("Block not found")
		return
	}

	fmt.Println("Desired Block:", desiredBlock)

}
func blockNumberExists(blockNumber int) bool {
	file, err := os.Open("block.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return false
	}
	defer file.Close()
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return false
	}
	blocks := []Block{}
	for _, line := range strings.Split(string(contents), "\n") {
		if line == "" {
			continue // Skip empty lines
		}

		block := Block{}
		err := json.Unmarshal([]byte(line), &block)
		if err != nil {
			fmt.Println("Error unmarshalling block:", err)
			return false
		}

		blocks = append(blocks, block)
	}
	desiredBlockNumber := blockNumber // Replace with the desired block number

	// var desiredBlock Block
	for _, block := range blocks {
		if block.BlockNumber == desiredBlockNumber {

			return true
		}
	}

	return false

}

func calculateBlockHash(block Block) string {
	blockBytes, _ := json.Marshal(block)
	hashBytes := sha256.Sum256(blockBytes)
	return fmt.Sprintf("%x", hashBytes)
}

func main() {

	db, err := leveldb.OpenFile("database", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 	for i := 1; i <= 1000; i++ {
	// 		txn := transaction{
	// 			Key: fmt.Sprintf("SIM%d", i),
	// 			Data: struct {
	// 				Val int
	// 				Ver float64
	// 			}{
	// 				Val: i,
	// 				Ver: 1.0,
	// 			},
	// 		}
	// 		data, err := json.Marshal(txn)
	// 		if err != nil {
	// 			log.Println("error encoding transaction", err)
	// 			continue
	// 		}
	// 		err = db.Put([]byte(txn.Key), data, nil)
	// 		if err != nil {
	// 			log.Println("Error storing the transaction", err)
	// 		}

	// 	}

	// }

	// populate LevelDB with 1000 entries
	for i := 1; i <= 10; i++ {
		key := fmt.Sprintf("SIM%d", i)
		data := value{
			Val:     float64(i),
			Version: 1.0,
		}
		txn := Transaction{
			Key:  key,
			Data: data,
		}
		jsonData, err := json.Marshal(txn)
		if err != nil {
			log.Fatal(err)
		}
		err = db.Put([]byte(key), jsonData, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
	// fmt.Println(txn.Data.val)
	// fmt.Println(txn.Data.version)
	// for i := 1; i <= 10; i++ {
	// 	key := fmt.Sprintf("SIM%d", i)
	// 	valueBytes, err := db.Get([]byte(key), nil)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	var txn Transaction
	// 	err = json.Unmarshal(valueBytes, &txn)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	fmt.Printf("Transaction with key %s: %+v\n", key, txn)
	// }

	input := `[
    {"key":"SIM1","data" : {"val": 2, "ver": 1.0}},
	{"key":"SIM2","data" : {"val": 3, "ver": 2.0}},
	{"key":"SIM3","data" : {"val": 4, "ver": 2.0}},
	{"key":"SIM4","data" : {"val": 5, "ver": 1.0}}
	]`

	var Txns []Transaction
	err = json.Unmarshal([]byte(input), &Txns)
	if err != nil {
		log.Fatal("error sorry", err)
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(Txns))

	for i := 0; i < len(Txns); i++ {
		go calculateTransactionHash(&Txns[i], wg)
	}

	wg.Wait()

	filePath := "blocks"
	blockSize := 2
	myChan := make(chan Block)
	go addTransactionToBlock(Txns, db, blockSize, myChan)
	blocks := []Block{}
	index := 0
	for block := range myChan {
		if block.BlockNumber == 1 {
			block.PrevBlockHash = "0xabc123"
			block.BlockHash = calculateBlockHash(block)

		} else {
			block.PrevBlockHash = blocks[index-1].BlockHash
			block.BlockHash = calculateBlockHash(block)

		}
		blocks = append(blocks, block)
		index++

		addBlockToFile(filePath, block)

	}

	for {
		fmt.Println("choose the number of your choice")
		fmt.Println("1.enter the number to get the block")
		fmt.Println("2.To get the content of all blocks")

		var num int
		_, err := fmt.Scanln(&num)
		if err != nil {
			fmt.Println("invalid input")
			continue
		}
		switch num {
		case 1:
			fmt.Println("Enter the block Number you wanted to see")
			var input1 int
			_, err := fmt.Scanln(&input1)
			if err != nil {
				fmt.Println("Invalid input")
				continue
			}
			findByBlockNumber(input1)
		case 2:
			GetAllBlocksFromFile(filePath)

		case 3:
			fmt.Println("Exiting from the program")
			return

		default:
			fmt.Println("Number is not 1, 2, or 3.")
		}

	}

}
