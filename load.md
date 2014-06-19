listing all queues ( without the quotes )
=========================================

store event queues
------------------

```
curl -u guest:guest http://localhost:15672/api/queues -s | jq '.[].name' | grep -E 'stores.*events' | sed 's/"//g' > stores.events.queues
```

store box queues
----------------

```
curl -u guest:guest http://localhost:15672/api/queues -s | jq '.[].name' | grep -E 'stores.*boxes' | sed 's/"//g' > stores.boxes.queues
```

delete queue contents
=====================

```
cat stores.boxes.queues | ( while read q; do curl -u guest:guest -iX DELETE http://localhost:15672/api/queues/%2f/$q/contents; done )
```

basic test
==========

1 message on 1 queue

```
head -1 ~/files/stores.events.queues | ( while read q; do ./amqpc -n=1 -p '' $q < ~/files/event.1.json; done )
```

load
====

15 boxes on each store boxes queue
----------------------------------

/!\ -g does not give accurate results with -n /!\


```
cat ~/files/stores.boxes.queues | ( while read q; do ./amqpc -n=15 -i=1 -p '' $q < ~/files/box.31.json &> /dev/null; done )
```

15 events on each store events queue
-------------------------------------

in a second terminal

```
cat ~/files/stores.events.queues | ( while read q; do ./amqpc -n=15 -i=1 -p '' $q < ~/files/event.31.json &> /dev/null; done )
```

1000 event.1450.json on central queue
-------------------------------------

```
./amqpc -n=1000 -i=1 -p '' central.events < ~/files/event.1450.json &> /dev/null
```

22000 event.31.json on central event queue
------------------------------------------

```
./amqpc -n=22000 -i=1 -p '' central.events < ~/files/event.31.json &> /dev/null
```

420000 event.1.json on central event queue
-------------------------------------------

this time, do it differently :), does not require that precision


```
for i in {1..5}; do ./amqpc -n=10000 -i=1 -g=10 -p '' central.events < ~/files/event.1.json &> /dev/null; done
```

( takes approx a minute )
