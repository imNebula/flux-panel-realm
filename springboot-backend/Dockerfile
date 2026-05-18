# syntax=docker/dockerfile:1.7
# 构建阶段
FROM --platform=$BUILDPLATFORM maven:3.9.6-eclipse-temurin-21 AS build
WORKDIR /app

ARG MAVEN_MIRROR_URL=https://maven.aliyun.com/repository/public
RUN mkdir -p /root/.m2 \
    && printf '%s\n' \
      '<settings>' \
      '  <mirrors>' \
      '    <mirror>' \
      '      <id>configured-mirror</id>' \
      '      <mirrorOf>*</mirrorOf>' \
      "      <url>${MAVEN_MIRROR_URL}</url>" \
      '    </mirror>' \
      '  </mirrors>' \
      '</settings>' > /root/.m2/settings.xml

COPY pom.xml .
RUN --mount=type=cache,target=/root/.m2/repository \
    mvn -B -ntp -DskipTests dependency:go-offline

COPY src ./src
RUN --mount=type=cache,target=/root/.m2/repository \
    mvn -B -ntp clean package -DskipTests

# 运行阶段
FROM --platform=$TARGETPLATFORM eclipse-temurin:21-jre
WORKDIR /app
ENV JAVA_OPTS="-Xmx512m -Xms256m -Dfile.encoding=UTF-8 -Duser.timezone=Asia/Shanghai"
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

RUN sed -i 's/archive.ubuntu.com/mirrors.ustc.edu.cn/g' /etc/apt/sources.list \
    && sed -i 's/security.ubuntu.com/mirrors.ustc.edu.cn/g' /etc/apt/sources.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends fontconfig fonts-dejavu \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /app/target/*.jar app.jar
EXPOSE 6365

ENTRYPOINT ["sh", "-c", "exec java $JAVA_OPTS -jar /app/app.jar"]
