# Setup some env variables
export ENDPOINT=http://rulesengine-sample-rulio.apps-crc.testing
echo  $ENDPOINT
export LOCATION=here
echo $LOCATION

# Write a fact.
echo "---creating a fact---"
curl -s -d 'fact={"have":"tacos"}' "$ENDPOINT/api/loc/facts/add?location=$LOCATION"

# Query for the fun of it.
echo "---querying fact---"
curl -s -d 'pattern={"have":"?x"}' "$ENDPOINT/api/loc/facts/search?location=$LOCATION" | \
  python -mjson.tool

# Write a simple rule.
echo "---creating a rule"
cat <<EOF | curl -s -d "@-" "$ENDPOINT/api/loc/rules/add?location=$LOCATION"
{"rule": {"when":{"pattern":{"wants":"?x"}},
          "condition":{"pattern":{"have":"?x"}},
          "action":{"code":"var msg = 'eat ' + x; console.log(msg); msg;"}}}
EOF

# Send an event.
echo "---sending an event---"
curl -d 'event={"wants":"tacos"}' "$ENDPOINT/api/loc/events/ingest?location=$LOCATION" | \
   python -mjson.tool

# Get Metrics
echo "---location stats---"
curl -s "$ENDPOINT/api/loc/admin/stats?location=$LOCATION" | python -mjson.tool

echo "---overall stats---"
curl -s "$ENDPOINT/api/sys/stats" | python -mjson.tool
