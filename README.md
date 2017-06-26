# Doctor Route 

## About
Simple app to test zero downtime deployments of CF and Routing release.

## Instructions to deploy as a TCP application.
1. Follow instructions [here](https://github.com/cloudfoundry-incubator/routing-release#post-deploy-steps) to enable TCP routing in your CF deployment. 
1. Run the following commands

  ```bash
  git clone https://github.com/cf-routing/doctorroute.git
  cd doctorroute
  cf api api.domain.com
  cf auth [your_username] [your_password]
  cf create-org testorg
  cf target -o testorg
  cf create-space testspace
  cf target -o testorg -s testspace
  cf push tcpapp -d tcp.domain.com --random-route
  ```

3. After staging the app successfully, run the command `cf app tcpapp` to get the URL for the app.

  ```bash
  ./test.sh tcp.domain.com TCP_PORT /health| telnet
 
  HTTP/1.1 200 OK
  Date: Wed, 21 Sep 2016 21:54:05 GMT
  Content-Length: 36
  Content-Type: text/plain; charset=utf-8
  
  {"TotalRequests":0,"Responses":null}
```

**Note**: To stage TCP application with a different port run `CF_TRACE=true cf router-groups` to list the reservable port range. Push the with no route `cf push tcpapp -d tcp.domain.com --no-route` and map the route with chosen port later `cf map-route doctorroute --port [chosen_port]`.
