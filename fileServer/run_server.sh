
./http_static_file_server -d ./public_root -auth ./auth.yaml -cert cert/server.crt -cert-key cert/server.key -p 3000 &> server.log &

echo $! > server.pid