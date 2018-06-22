# Deluxe One Nats. Improved nats client for java

## Prerequisites
* jdk 1.8. Check `java -version`
* mvn 3.5+. Check `mvn -version`

## How to build
* Execute `mvn clean package`
* You should see output similar to this:
```
[INFO] --- maven-jar-plugin:2.4:jar (default-jar) @ nats ---
[INFO] Building jar: /Users/aleksey/Development/deluxe/one-nats/java/target/nats-1.0.0.jar
[INFO] ------------------------------------------------------------------------
[INFO] BUILD SUCCESS
[INFO] ------------------------------------------------------------------------
[INFO] Total time: 1.942 s
[INFO] Finished at: 2018-06-15T12:02:53+03:00
[INFO] ------------------------------------------------------------------------
```

## Push to Github maven repo

Generate your personal access token for Github:
* Log in into Github
* Go to `Settings -> Developer settings -> Personal access tokens`
* Generate a new token
* Give access to `repo` and `user` scopes
* Press `Generate token` button 
* Copy the generated token. DO NOT PASS it over the Internet!

Edit your local `~/.m2/settings.xml`, add the following: 
 
```
<servers>
  <server>
    <id>github</id>
    <password>Your token generated at previous step</password>
  </server>
</servers>
```

Make sure you added the generated token into `password` xml element.

## Deploy to Github maven repo
Execute `mvn deploy`

Once done, you should see output similar to this:
```
[INFO]
[INFO] --- site-maven-plugin:0.11:site (default) @ nats ---
[INFO] Creating 9 blobs
[INFO] Creating tree with 10 blob entries
[INFO] Creating commit with SHA-1: 58c5bc9044193f79970aa60c18303ffc23f7b14d
[INFO] Creating reference refs/heads/mvn-repo starting at commit 58c5bc9044193f79970aa60c18303ffc23f7b14d
[INFO] ------------------------------------------------------------------------
[INFO] BUILD SUCCESS
[INFO] ------------------------------------------------------------------------
[INFO] Total time: 8.439 s
[INFO] Finished at: 2018-06-15T12:06:56+03:00
[INFO] ------------------------------------------------------------------------
```

Also you should be able to see the artifact in the Github repo `https://github.com/d3sw/one-nats/tree/mvn-repo`

## How to use Github maven repo
Please see `../test` java project with example 