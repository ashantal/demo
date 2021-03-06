/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright stateship.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// ============================================================================================================================
// Read - read a generic variable from ledger
//
// Shows Off GetState() - reading a key/value from the ledger
//
// Inputs - Array of strings
//  0
//  key
//  "abc"
//
// Returns - string
// ============================================================================================================================
func read(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var key, jsonResp string
	var err error
	fmt.Println("starting read")

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting key of the var to query")
	}

	// input sanitation
	err = sanitize_arguments(args)
	if err != nil {
		return shim.Error(err.Error())
	}

	key = args[0]
	valAsbytes, err := stub.GetState(key) //get the var from ledger
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + key + "\"}"
		return shim.Error(jsonResp)
	}

	fmt.Println("- end read")
	return shim.Success(valAsbytes) //send it onward
}

// ============================================================================================================================
// Get everything we need (states + CC Demo + companies)
//
// Inputs - none
//
// Returns:
// {
//	"states": [{
//			"id": "o99999999",
//			"company": "United listings"
//			"username": "alice"
//	}],
//	"listings": [{
//		"id": "m1490898165086",
//		"color": "white",
//		"docType" :"listing",
//		"state": {
//			"company": "United listings"
//			"username": "alice"
//		},
//		"size" : 35
//	}]
// }
// ============================================================================================================================
func read_everything(stub shim.ChaincodeStubInterface) pb.Response {
	type Everything struct {
		States   []State   `json:"states"`
		Listings []Listing `json:"listings"`
	}
	var everything Everything

	// ---- Get All Listings ---- //
	resultsIterator, err := stub.GetStateByRange("l0", "l9999999999999999999")
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	for resultsIterator.HasNext() {
		aKeyValue, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		queryKeyAsStr := aKeyValue.Key
		queryValAsBytes := aKeyValue.Value
		fmt.Println("on listing id - ", queryKeyAsStr)
		var listing Listing
		json.Unmarshal(queryValAsBytes, &listing)                  //un stringify it aka JSON.parse()
		everything.Listings = append(everything.Listings, listing) //add this listing to the list
	}
	fmt.Println("listing array - ", everything.Listings)

	// ---- Get All States ---- //
	statesIterator, err := stub.GetStateByRange("s0", "s9999999999999999999")
	if err != nil {
		return shim.Error(err.Error())
	}
	defer statesIterator.Close()

	for statesIterator.HasNext() {
		aKeyValue, err := statesIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		queryKeyAsStr := aKeyValue.Key
		queryValAsBytes := aKeyValue.Value
		fmt.Println("on state id - ", queryKeyAsStr)
		var state State
		json.Unmarshal(queryValAsBytes, &state) //un stringify it aka JSON.parse()

		if state.Enabled { //only return enabled states
			everything.States = append(everything.States, state) //add this listing to the list
		}
	}
	fmt.Println("state array - ", everything.States)

	//change to array of bytes
	everythingAsBytes, _ := json.Marshal(everything) //convert to array of bytes
	return shim.Success(everythingAsBytes)
}

// ============================================================================================================================
// Get history of asset
//
// Shows Off GetHistoryForKey() - reading complete history of a key/value
//
// Inputs - Array of strings
//  0
//  id
//  "m01490985296352SjAyM"
// ============================================================================================================================
func getHistory(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	type AuditHistory struct {
		TxId  string  `json:"txId"`
		Value Listing `json:"value"`
	}
	var history []AuditHistory
	var listing Listing

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	listingId := args[0]
	fmt.Printf("- start getHistoryForListing: %s\n", listingId)

	// Get History
	resultsIterator, err := stub.GetHistoryForKey(listingId)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	for resultsIterator.HasNext() {
		historyData, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		var tx AuditHistory
		tx.TxId = historyData.TxId                  //copy transaction id over
		json.Unmarshal(historyData.Value, &listing) //un stringify it aka JSON.parse()
		if historyData.Value == nil {               //listing has been deleted
			var emptyListing Listing
			tx.Value = emptyListing //copy nil listing
		} else {
			json.Unmarshal(historyData.Value, &listing) //un stringify it aka JSON.parse()
			tx.Value = listing                          //copy listing over
		}
		history = append(history, tx) //add this tx to the list
	}
	fmt.Printf("- getHistoryForListing returning:\n%s", history)

	//change to array of bytes
	historyAsBytes, _ := json.Marshal(history) //convert to array of bytes
	return shim.Success(historyAsBytes)
}

// ============================================================================================================================
// Get history of asset - performs a range query based on the start and end keys provided.
//
// Shows Off GetStateByRange() - reading a multiple key/values from the ledger
//
// Inputs - Array of strings
//       0     ,    1
//   startKey  ,  endKey
//  "listings1" , "listings5"
// ============================================================================================================================
func getListingsByRange(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	startKey := args[0]
	endKey := args[1]

	resultsIterator, err := stub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		aKeyValue, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		queryResultKey := aKeyValue.Key
		queryResultValue := aKeyValue.Value

		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResultKey)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResultValue))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getListingByRange queryResult:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

/*{
	"selector": {
	  "data.state.state_name": {
		"$eq": "kitchen upgraded"
	  }
	}
}
*/
func getQueryResultForQueryString(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	queryString := args[0]
	fmt.Printf("- getQueryResultForQueryString queryString:\n%s\n", queryString)
	resultsIterator, err := stub.GetQueryResult(queryString)
	defer resultsIterator.Close()
	if err != nil {
		return shim.Error(err.Error())
	}
	// buffer is a JSON array containing QueryRecords
	var buffer bytes.Buffer
	buffer.WriteString("[")
	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse,
			err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
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
	fmt.Printf("- getQueryResultForQueryString queryResult:\n%s\n", buffer.String())
	return shim.Success(buffer.Bytes())
}
