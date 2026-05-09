# 注册
curl -X POST http://localhost:8001/api/users/register \
  -H "Content-Type: application/json" \
  -d '{"username":"tom","email":"tom@example.com","password":"123456"}'
# {"code":201,"message":"注册成功","data":{"id":"xxx","username":"tom","email":"tom@example.com"}}

# 登录
curl -X POST http://localhost:8001/api/users/login \
  -H "Content-Type: application/json" \
  -d '{"username":"tom","password":"123456"}'
# {"code":200,"message":"登录成功","data":{"token":"eyJhbGci..."}}

# 获取用户信息（用返回的 id 替换）
curl http://localhost:8001/api/users/{id}

# 重复注册
curl -X POST http://localhost:8001/api/users/register \
  -H "Content-Type: application/json" \
  -d '{"username":"tom","email":"tom2@example.com","password":"123456"}'
# {"code":400,"message":"用户名已存在"}

# 错误密码
curl -X POST http://localhost:8001/api/users/login \
  -H "Content-Type: application/json" \
  -d '{"username":"tom","password":"wrong"}'
# {"code":401,"message":"用户名或密码错误"}