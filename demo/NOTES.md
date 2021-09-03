# Notes
To run use command:
`revealgo --theme league PRESENTATION.md`


## Order of Operations
1. Make a new project
2. Export the below
```shell
export ENDPOINT=http://rulesengine-sample-rulio.apps-crc.testing

echo  $ENDPOINT

export LOCATION=here

echo $LOCATION
```
3. Have one terminal watch routes `oc get deployments --all-namespaces -w`
4. Have one terminal watch pods `oc get pods --all-namespaces -w`
5. Have one terminal watch services `oc get services --all-namespaces -w`
6. Have one terminal watch routes `oc get routes --all-namespaces -w`
7. Run `./demo.sh` script
8. Run `/.curl.sh` script

## Curls
Setup Curls
```Shell
# Write a fact.
curl -s -d 'fact={"have":"tacos"}' "$ENDPOINT/api/loc/facts/add?location=$LOCATION"

# Query for the fun of it.
curl -s -d 'pattern={"have":"?x"}' "$ENDPOINT/api/loc/facts/search?location=$LOCATION" | \
  python -mjson.tool

# Write a simple rule.
cat <<EOF | curl -s -d "@-" "$ENDPOINT/api/loc/rules/add?location=$LOCATION"
{"rule": {"when":{"pattern":{"wants":"?x"}},
          "condition":{"pattern":{"have":"?x"}},
          "action":{"code":"var msg = 'eat ' + x; console.log(msg); msg;"}}}
EOF

# Send an event.
curl -d 'event={"wants":"tacos"}' "$ENDPOINT/api/loc/events/ingest?location=$LOCATION" | \
   python -mjson.tool
```

Metrics Curls
```Shell
curl -s "$ENDPOINT/api/loc/admin/stats?location=$LOCATION" | python -mjson.tool
curl -s "$ENDPOINT/api/sys/stats" | python -mjson.tool
```