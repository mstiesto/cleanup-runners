#! /bin/bash
### execution example: ./clean-up.sh runner-01

RUNNER=$1
STATUS=$?
FILESYSTEM=/dev/vda1
CAPACITY=85
GITLAB_API=https://gitlab.example.com/api/v4/runners
GITLAB_TOKEN=xxxxxx
ELEMENT_API=https://element.example.com/
ELEMENT_TOKEN=xxxxxx

declare -A runner_ids=(["runner-01"]=xx ["runner-02"]=xx ["runner-03"]=xx)
for runner in "${!runner_ids[@]}"
do
    if [[ `echo $RUNNER | tr '[:upper:]' '[:lower:]'` = "$runner" ]];
    then
        RUNNER_ID=${runner_ids[$runner]}
    fi
done
if [[ -z $RUNNER_ID ]];
then
    echo "Runner name is not valid. Should be like runner-xx"
    exit 1
fi

echo -e "--> Script started on $RUNNER with ID $RUNNER_ID\n--> $(date)"
if [ $(df -P $FILESYSTEM | awk '{ gsub("%",""); capacity = $5 }; END { print capacity }') -gt $CAPACITY ]; 
then
        RESPONSE=1
        until [ "$RESPONSE" -eq 200 ]
        do
                sleep 5
                echo "--> Pausing the $RUNNER ..."
                RESPONSE=$(curl --write-out '%{http_code}'\
                --silent --location --request PUT $GITLAB_API/$RUNNER_ID \
                --form "active=false" \
                --header "PRIVATE-TOKEN: $GITLAB_TOKEN" --output /dev/null)
		echo "--> response: $RESPONSE"
        done
        RESPONSE=1
        until [ "$RESPONSE" -eq 200 ]
        do
                sleep 5
                echo "--> Sending pause message to the element ..."
                RESPONSE=$(curl --write-out '%{http_code}'\
                --silent --location --request POST $ELEMENT_API \
                --header "Content-Type: application/json" \
                --data-raw "{ \"text\": \"$RUNNER paused\", \"displayName\": \"runners-cleanup\"}" \
                --output /dev/null)
		echo "--> response: $RESPONSE"
        done
        JOBS=1
        until [ "$JOBS" -eq 0 ]
        do
                sleep 5
                echo "--> Waiting for jobs to be finished ..."
                JOBS=$(curl --location --silent\
                --request GET $GITLAB_API/$RUNNER_ID/jobs?status=running \
                --header "PRIVATE-TOKEN: $GITLAB_TOKEN" | jq length)
                echo "--> jobs count: $JOBS"
        done
        STATUS=1
        until [ $STATUS -eq 0 ]
        do
                sleep 5
                echo "--> Runner paused and jobs finished, running prune script ..."
                /opt/docker-prune.sh
                STATUS=$?
        done
        RESPONSE=1
        until [ "$RESPONSE" -eq 200 ]
        do
                sleep 5
                echo "--> Starting the $RUNNER ..."
                RESPONSE=$(curl --write-out '%{http_code}'\
                --silent --location --request PUT $GITLAB_API/$RUNNER_ID \
                --form "active=true" \
                --header "PRIVATE-TOKEN: $GITLAB_TOKEN" --output /dev/null)
                echo "--> response: $RESPONSE"
        done
        RESPONSE=1
        until [ "$RESPONSE" -eq 200 ]
        do
                sleep 5
                echo "--> Sending continue message to the element ..."
                RESPONSE=$(curl --write-out '%{http_code}'\
                --silent --location --request POST $ELEMENT_API \
                --header "Content-Type: application/json" \
                --data-raw "{ \"text\": \"$RUNNER continued\", \"displayName\": \"runners-cleanup\"}" \
                --output /dev/null)
		echo "--> response: $RESPONSE"
        done
else 
        echo -e "################################### \nThe storage is not yet full enough \n###################################"
fi
echo -e "--> $(date)\n--> Fiinished"
