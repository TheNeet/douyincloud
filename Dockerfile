FROM public-cn-beijing.cr.volces.com/public/golang:alpine as builder
# 指定构建过程中的工作目录
WORKDIR /app
# 将当前目录（dockerfile所在目录）下所有文件都拷贝到工作目录下（.dockerignore中文件除外）
COPY . /app/

# 执行代码编译命令。操作系统参数为linux，编译后的二进制产物命名为main，并存放在当前目录下。
RUN GOPROXY=https://goproxy.cn,direct GOOS=linux GOARCH=amd64 go build -o douyincloud .

FROM public-cn-beijing.cr.volces.com/public/dycloud-golang:alpine-3.17

WORKDIR /opt/application

COPY --from=builder /app /opt/application

USER root

RUN apk add ca-certificates curl
RUN mv /opt/application/douyincloud_egress.crt /usr/local/share/ca-certificates/douyincloud_egress.crt
RUN update-ca-certificates

RUN chmod 777 run.sh
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

EXPOSE 8000

#CMD /opt/application/run.sh