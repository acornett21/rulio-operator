# all the commands to setup and run this operator
cd ..
pwd

echo "---creating new project---"
oc new-project rulio

echo "---running make install---"
make install

echo "---running kubectl apply---"
kubectl apply -f config/samples/rules_v1alpha1_rulesengine.yaml

echo "---running make run---"
make run
