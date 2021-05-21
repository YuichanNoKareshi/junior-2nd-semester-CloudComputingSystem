package mapreduce

import (
	"os"
	// "hash/fnv"
    // "io/ioutil"
	"encoding/json"
	"sort"
)

type KVSlice [] KeyValue
 
func (a KVSlice) Len() int {    // rewite Len()
	return len(a)
}

func (a KVSlice) Swap(i, j int){     // rewrite Swap()
	a[i], a[j] = a[j], a[i]
}

func (a KVSlice) Less(i, j int) bool {    // rewrite Less()
    return a[j].Key < a[i].Key 
}

func doReduce(
	jobName string, // the name of the whole MapReduce job
	reduceTask int, // which reduce task this is
	outFile string, // write the output here
	nMap int, // the number of map tasks that were run ("M" in the paper)
	reduceF func(key string, values []string) string,
) {
	//
	// doReduce manages one reduce task: it should read the intermediate
	// files for the task, sort the intermediate key/value pairs by key,
	// call the user-defined reduce function (reduceF) for each key, and
	// write reduceF's output to disk.
	//
	// You'll need to read one intermediate file from each map task;
	// reduceName(jobName, m, reduceTask) yields the file
	// name from map task m.
	//
	// Your doMap() encoded the key/value pairs in the intermediate
	// files, so you will need to decode them. If you used JSON, you can
	// read and decode by creating a decoder and repeatedly calling
	// .Decode(&kv) on it until it returns an error.
	//
	// You may find the first example in the golang sort package
	// documentation useful.
	//
	// reduceF() is the application's reduce function. You should
	// call it once per distinct key, with a slice of all the values
	// for that key. reduceF() returns the reduced value for that key.
	//
	// You should write the reduce output as JSON encoded KeyValue
	// objects to the file named outFile. We require you to use JSON
	// because that is what the merger than combines the output
	// from all the reduce tasks expects. There is nothing special about
	// JSON -- it is just the marshalling format we chose to use. Your
	// output code will look something like this:
	//
	// enc := json.NewEncoder(file)
	// for key := ... {
	// 	enc.Encode(KeyValue{key, reduceF(...)})
	// }
	// file.Close()
	//
	// Your code here (Part I).
	//

	var kvpairs []KeyValue

	for m := 0; m < nMap; m++{
		interFileName := reduceName(jobName, m, reduceTask)

		interFd,_ := os.Open(interFileName)
		dec := json.NewDecoder(interFd)

		for {
			var kv KeyValue
			err := dec.Decode(&kv)
			if err != nil{
				break
			}
			kvpairs = append(kvpairs, kv)
		}
		interFd.Close()
		
	}
	sort.Sort(sort.Reverse(KVSlice(kvpairs))) // sort from low to high

	outFd,_ := os.OpenFile(outFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	enc := json.NewEncoder(outFd)
	
	var values [] string
	var currKey string
	for i := range kvpairs{
		if i == 0 {
			currKey = kvpairs[i].Key
			values = append(values, kvpairs[i].Value)
		} else if currKey == kvpairs[i].Key { // if Key not changes, append value
			values = append(values, kvpairs[i].Value)
		} else { // if Key changes, call reduceF, reset values and change currKey
			sort.Strings(values)
			tempValue := reduceF(currKey, values)
			tempPairs := KeyValue{currKey, tempValue}
			enc.Encode(&tempPairs)

			values = values[0:0]
			values = append(values, kvpairs[i].Value)

			currKey = kvpairs[i].Key
		}
	}
	sort.Strings(values)
	tempValue := reduceF(currKey, values)
	tempPairs := KeyValue{currKey, tempValue}
	enc.Encode(&tempPairs)

	outFd.Close()	

}
