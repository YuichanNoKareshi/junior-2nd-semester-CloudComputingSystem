date
for i in `seq 1 10`
do
{
    go run ./gfs_api.go ./json_struct.go ./lock.go ./testMultiCreate.go
}&
done
wait
date 
for i in `seq 1 10`
do
{
    go run ./gfs_api.go ./json_struct.go ./lock.go ./testMultiWrite.go
}&
done
wait
date
for i in `seq 1 10`
do
{
    go run ./gfs_api.go ./json_struct.go ./lock.go ./testMultiRead.go
}&
done
wait
date