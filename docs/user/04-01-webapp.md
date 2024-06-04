1. Connect to cluster with BTP Operator
2. Change directory to BTP Manager source code
3. Change branch to sm-integration 
4. Open 2 terminal windows
   In one - Run make app
   In second - Run make ui

If any of pors is taken u can use to clean:
kill -9 $(lsof -i tcp:8081 -t)