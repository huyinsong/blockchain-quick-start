// Written by Xu Chen Hao
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/chaincode/shim/ext/statebased"
	sc "github.com/hyperledger/fabric/protos/peer"
	"strconv"
	"time"
)

var logger = shim.NewLogger("chaincode")

// Define the Smart Contract structure
type SmartContract struct {
}

type R_Err struct {
	Reason string `json:"reason"`
}

type Manifest struct {
	Shipper           string `json:"shipper"`
	Consignee         string `json:"consignee"`
	FromPort          string `json:"fromPort"`
	DespPortCode      string `json:"despPortCode"`
	DespPort          string `json:"despPort"`
	Freight           string `json:"freight"`
	DistinatePortCode string `json:"distinatePortCode"`
	MasterBillNo      string `json:"masterBillNo"`
	ToPort            string `json:"toPort"`
	DistinatePort     string `json:"distinatePort"`
	Pack              string `json:"pack"`
	LanguageIdentity  string `json:"languageIdentity"`
	Ata               string `json:"ata"`
	Atd               string `json:"atd"`
	Carrier           string `json:"carrier"`
	GrossWeight       string `json:"grossWeight"`
	NetWeight         string `json:"netWeight"`
	FlightNo          string `json:"flightNo"`
	Measure           string `json:"measure"`
	ToPortCode        string `json:"toPortCode"`
	Mark              string `json:"mark"`
	PackNo            string `json:"packNo"`
	WeightCode        string `json:"weightCode"`
}

func (s *SmartContract) returnError(reason string) sc.Response {

	var re R_Err

	re.Reason = reason
	logger.Error(re.Reason)
	reAsBytes, _ := json.Marshal(re)

	return shim.Success(reAsBytes)
}

func (s *SmartContract) Init(APIstub shim.ChaincodeStubInterface) sc.Response {
	return shim.Success(nil)
}

func (s *SmartContract) Invoke(APIstub shim.ChaincodeStubInterface) sc.Response {

	// Retrieve the requested Smart ContgetQueryResultForQueryStringract function and arguments
	function, args := APIstub.GetFunctionAndParameters()
	// Route to the appropriate handler function to interact with the ledger appropriately
	if function == "uploadManifest" {
		return s.uploadManifest(APIstub, args)
	} else if function == "queryManifest" {
		return s.queryManifest(APIstub, args)
	} else if function == "queryManifestHistory" {
		return s.queryManifestHistory(APIstub, args)
	} else if function == "richQueryManifest" {
		return s.richQueryManifest(APIstub, args)
	} else if function == "kepAddOrgs" {
		return s.kepAddOrgs(APIstub, args)
	} else if function == "kepDelOrgs" {
		return s.kepDelOrgs(APIstub, args)
	} else if function == "kepListOrgs" {
		return s.kepListOrgs(APIstub, args)
	} else if function == "delKEP" {
		return s.delKEP(APIstub, args)
	}

	return s.returnError("Invalid Smart Contract function name.")
}

func (s *SmartContract) kepAddOrgs(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) < 2 {
		return s.returnError("No orgs to add specified")
	}

	epKey := args[0]
	logger.Debug("Got key-level endorsement policy key: " + epKey)

	// get the endorsement policy for the key
	epBytes, err := stub.GetStateValidationParameter(epKey)
	if err != nil {
		return s.returnError("Error get endorsement policy: " + err.Error())
	}
	ep, err := statebased.NewStateEP(epBytes)
	if err != nil {
		return s.returnError("Error generate new endorsement policy: " + err.Error())
	}

	// add organizations to endorsement policy
	err = ep.AddOrgs(statebased.RoleTypePeer, args[1:]...)
	if err != nil {
		return s.returnError("Error add organizations to endorsement policy: " + err.Error())
	}
	epBytes, err = ep.Policy()
	if err != nil {
		return s.returnError("Error generate endorsement policy bytes: " + err.Error())
	}

	// set the modified endorsement policy for the key
	err = stub.SetStateValidationParameter(epKey, epBytes)
	if err != nil {
		return s.returnError("Error set key level endorsement policy: " + err.Error())
	}

	return shim.Success(nil)
}


func (s *SmartContract) kepDelOrgs(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) < 2 {
		return s.returnError("No orgs to delete specified")
	}

	epKey := args[0]
	logger.Debug("Got key-level endorsement policy key: " + epKey)

	// get the endorsement policy for the key
	epBytes, err := stub.GetStateValidationParameter(epKey)
	if err != nil {
		return s.returnError("Error get endorsement policy: " + err.Error())
	}
	ep, err := statebased.NewStateEP(epBytes)
	if err != nil {
		return s.returnError("Error generate new endorsement policy: " + err.Error())
	}

	// delete organizations from the endorsement policy of that key
	ep.DelOrgs(args[1:]...)
	epBytes, err = ep.Policy()
	if err != nil {
		return s.returnError("Error generate endorsement policy bytes: " + err.Error())
	}

	// set the modified endorsement policy for the key
	err = stub.SetStateValidationParameter(epKey, epBytes)
	if err != nil {
		return s.returnError("Error set key level endorsement policy: " + err.Error())
	}

	return shim.Success(nil)
}

// listOrgs returns the list of organizations currently part of
// the state's endorsement policy
func (s *SmartContract) kepListOrgs(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 1 {
		return shim.Error("No key specified or too many keys specified")
	}

	epKey := args[0]
	logger.Debug("Got key-level endorsement policy key: " + epKey)

	// get the endorsement policy for the key
	epBytes, err := stub.GetStateValidationParameter(epKey)
	if err != nil {
		return s.returnError("Error get endorsement policy: " + err.Error())
	}
	ep, err := statebased.NewStateEP(epBytes)
	if err != nil {
		return s.returnError("Error generate new endorsement policy: " + err.Error())
	}

	// get the list of organizations in the endorsement policy
	orgs := ep.ListOrgs()
	orgsList, err := json.Marshal(orgs)
	if err != nil {
		return s.returnError("Error marshal orgs json: " + err.Error())
	}

	return shim.Success(orgsList)
}

// delEP deletes the state-based endorsement policy for the key altogether
func (s *SmartContract) delKEP(stub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 1 {
		return shim.Error("No key specified or too many keys specified")
	}

	epKey := args[0]
	logger.Debug("Got key-level endorsement policy key: " + epKey)

	// set the modified endorsement policy for the key to nil
	err := stub.SetStateValidationParameter(epKey, nil)
	if err != nil {
		return s.returnError("Error set key level endorsement policy: " + err.Error())
	}

	return shim.Success(nil)
}


func (s *SmartContract) validate(manifest Manifest) bool {

	//TODO define some validate conditions
	return true
}

func (s *SmartContract) writeChain(APIstub shim.ChaincodeStubInterface, manifest Manifest) error {
	manifestAsBytes, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	logger.Debug("Write chain: " + string(manifestAsBytes))
	err = APIstub.PutState(manifest.MasterBillNo, manifestAsBytes)
	if err != nil {
		return err
	}
	return nil
}

func (s *SmartContract) uploadManifest(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 1 {
		return s.returnError("参数数量不正确")
	}

	logger.Debug("获取请求参数: " + args[0])

	manifestAsBytes := []byte(args[0])
	var manifest Manifest
	err := json.Unmarshal(manifestAsBytes, &manifest)
	if err != nil {
		return s.returnError("主舱单格式错误: " + err.Error())
	}

	// 验证主舱单是否合法
	if !s.validate(manifest) {
		return s.returnError("主舱单不合法")
	}

	// 数据上链
	err = s.writeChain(APIstub, manifest)
	if err != nil {
		return s.returnError("主舱单上链失败: " + err.Error())
	}

	return shim.Success(manifestAsBytes)

}

func (s *SmartContract) queryManifest(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 1 {
		return s.returnError("参数数量不正确")
	}

	masterBillNo := args[0]
	logger.Debug("Query on chain: " + masterBillNo)
	result, err := APIstub.GetState(masterBillNo)
	if err != nil {
		return s.returnError("主舱单查询失败" + err.Error())
	}
	return shim.Success(result)
}

func (s *SmartContract) richQueryManifest(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) == 1 {

		// rich query without pagination
		queryString := args[0]
		logger.Debug("Rich query on chain: " + queryString)
		result, err := s.richQuery(APIstub, queryString)
		if err != nil {
			return s.returnError("Rich query failed: " + err.Error())
		}
		return shim.Success(result)

	} else if len(args) == 3 {

		// rich query with pagination
		queryString := args[0]
		pageSize, err := strconv.ParseInt(args[1], 10, 32)
		if err != nil {
			return s.returnError("Error convert arg 2 to int32: " + err.Error())
		}
		bookmark := args[2]
		logger.Debugf("Rich query %s with pagination ( page size %s, bookmark %s )",
			queryString, pageSize, bookmark)
		result, err := s.richQueryWithPagination(APIstub, queryString, int32(pageSize), bookmark)
		if err != nil {
			return s.returnError("Rich query failed: " + err.Error())
		}
		return shim.Success(result)

	} else {
		return s.returnError("参数数量不正确")
	}
}

func (s *SmartContract) queryManifestHistory(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 1 {
		return s.returnError("参数数量不正确")
	}

	masterBillNo := args[0]
	logger.Debug("Query history on chain: " + masterBillNo)
	result, err := s.queryHistoryAsset(APIstub, masterBillNo)
	if err != nil {
		return s.returnError("主舱单查询失败" + err.Error())
	}
	return shim.Success(result)
}

func (s *SmartContract) queryHistoryAsset(stub shim.ChaincodeStubInterface, key string) ([]byte, error) {
	resultsIterator, err := stub.GetHistoryForKey(key)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing historic values for the marble
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"TxId\":")
		buffer.WriteString("\"")
		buffer.WriteString(response.TxId)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Value\":")
		// if it was a delete operation on given key, then we need to set the
		//corresponding value null. Else, we will write the response.Value
		//as-is (as the Value itself a JSON marble)
		if response.IsDelete {
			buffer.WriteString("null")
		} else {
			buffer.WriteString(string(response.Value))
		}

		buffer.WriteString(", \"Timestamp\":")
		buffer.WriteString("\"")
		buffer.WriteString(time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).String())
		buffer.WriteString("\"")

		buffer.WriteString(", \"IsDelete\":")
		buffer.WriteString("\"")
		buffer.WriteString(strconv.FormatBool(response.IsDelete))
		buffer.WriteString("\"")

		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	logger.Debug("- getHistory returning:\n%s\n", buffer.String())

	return buffer.Bytes(), nil
}

func (s *SmartContract) richQuery(stub shim.ChaincodeStubInterface, queryString string)([] byte, error) {

	logger.Debug("Get rich query request: \n%s\n", queryString)

	resultsIterator, err := stub.GetQueryResult(queryString)
	defer resultsIterator.Close()
	if err != nil {
		return nil, err
	}

	buffer, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (s *SmartContract) richQueryWithPagination(stub shim.ChaincodeStubInterface,
	queryString string, pageSize int32, bookmark string) ([] byte, error) {

	queryResults, err := getQueryResultForQueryStringWithPagination(stub, queryString, int32(pageSize), bookmark)
	if err != nil {
		return nil, err
	}
	return queryResults, nil
}

func getQueryResultForQueryStringWithPagination(stub shim.ChaincodeStubInterface,
	queryString string, pageSize int32, bookmark string) ([]byte, error) {

	logger.Debugf("- getQueryResultForQueryString queryString:\n%s\n", queryString)

	resultsIterator, responseMetadata, err := stub.GetQueryResultWithPagination(queryString, pageSize, bookmark)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	buffer, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return nil, err
	}

	bufferWithPaginationInfo := addPaginationMetadataToQueryResults(buffer, responseMetadata)

	logger.Debugf("- getQueryResultForQueryString queryResult:\n%s\n", bufferWithPaginationInfo.String())

	return buffer.Bytes(), nil
}

func constructQueryResponseFromIterator(resultsIterator shim.StateQueryIteratorInterface) (*bytes.Buffer, error) {
	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return &buffer, nil
}

func addPaginationMetadataToQueryResults(buffer *bytes.Buffer, responseMetadata *sc.QueryResponseMetadata) *bytes.Buffer {

	buffer.WriteString("[{\"ResponseMetadata\":{\"RecordsCount\":")
	buffer.WriteString("\"")
	buffer.WriteString(fmt.Sprintf("%v", responseMetadata.FetchedRecordsCount))
	buffer.WriteString("\"")
	buffer.WriteString(", \"Bookmark\":")
	buffer.WriteString("\"")
	buffer.WriteString(responseMetadata.Bookmark)
	buffer.WriteString("\"}}]")

	return buffer
}

// The main function is only relevant in unit test mode. Only included here for completeness.
func main() {

	// Create a new Smart Contract
	err := shim.Start(new(SmartContract))
	if err != nil {
		logger.Error("Error creating new Smart Contract: %s", err)
	}
}