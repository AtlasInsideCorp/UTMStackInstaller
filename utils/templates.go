package utils

const (
	composerFile  = "docker-compose.yml"
	probeTemplateLite = `version: "3"

volumes:
  postgres_data:
  wazuh_etc:
  wazuh_var:
  wazuh_logs:
  openvas_data:
  geoip_data:

networks:
  utmstack-net:

services:
  watchtower:
    container_name: watchtower
    restart: always
    image: containrrr/watchtower
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /root/.docker/config.json:/config.json
    environment:
      - WATCHTOWER_NO_RESTART=false
      - WATCHTOWER_POLL_INTERVAL=${UPDATES}

  logstash:
    container_name: logstash
    restart: always
    image: "utmstack.azurecr.io/logstash:${TAG}"
    volumes:
      - ${LOGSTASH_PIPELINE}:/usr/share/logstash/pipeline
      - /var/log/suricata:/var/log/suricata
      - wazuh_logs:/var/ossec/logs
      - ${CERT}:/cert
    ports:
      - 5044:5044
      - 8089:8089
      - 514:514
      - 514:514/udp
      - 2055:2055/udp
    environment:
      - CONFIG_RELOAD_AUTOMATIC=true
    networks:
      - utmstack-net

  datasources_mutate:
    container_name: datasources_mutate
    restart: always
    image: "utmstack.azurecr.io/datasources:${TAG}"
    volumes:
      - ${UTMSTACK_DATASOURCES}:/etc/utmstack
      - ${LOGSTASH_PIPELINE}:/usr/share/logstash/pipeline
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - SERVER_NAME
      - SERVER_TYPE
      - DB_HOST
      - DB_PASS
      - CORRELATION_URL
    networks:
      - utmstack-net
    command: ["python3", "-m", "utmstack.mutate"]

  datasources_transporter:
    container_name: datasources_transporter
    restart: always
    image: "utmstack.azurecr.io/datasources:${TAG}"
    volumes:
      - ${UTMSTACK_DATASOURCES}:/etc/utmstack
      - /var/log/suricata:/var/log/suricata
      - wazuh_etc:/var/ossec/etc
      - wazuh_var:/var/ossec/var
      - wazuh_logs:/var/ossec/logs
    environment:
      - SERVER_NAME
      - SERVER_TYPE
      - DB_HOST
      - DB_PASS
    networks:
      - utmstack-net
    command: ["python3", "-m", "utmstack.transporter"]

  datasources_probe_api:
    container_name: datasources_probe_api
    restart: always
    image: "utmstack.azurecr.io/datasources:${TAG}"
    volumes:
      - wazuh_etc:/var/ossec/etc
      - wazuh_var:/var/ossec/var
      - wazuh_logs:/var/ossec/logs
      - ${UTMSTACK_DATASOURCES}:/etc/utmstack
      - ${CERT}:/cert
    environment:
      - SERVER_NAME
      - SERVER_TYPE
      - DB_HOST
      - DB_PASS
      - SCANNER_IP
      - SCANNER_IFACE
    ports:
      - 23949:23949
      - 1514:1514
      - 1514:1514/udp
      - 1515:1515
      - 1516:1516
      - 55000:55000
    networks:
      - utmstack-net
    command: ["/pw.sh"]
`
	masterTemplate = `
  node1:
    container_name: node1
    restart: always
    image: "utmstack.azurecr.io/opendistro:${TAG}"
    ports:
      - "9200:9200"
    volumes:
      - ${ES_DATA}:/usr/share/elasticsearch/data
      - ${ES_BACKUPS}:/usr/share/elasticsearch/backups
    environment:
      - node.name=node1
      - discovery.seed_hosts=node1
      - cluster.initial_master_nodes=node1
      - "ES_JAVA_OPTS=-Xms${ES_MEM}g -Xmx${ES_MEM}g"
      - path.repo=/usr/share/elasticsearch
    networks:
      - utmstack-net

  postgres:
    container_name: postgres
    restart: always
    image: "utmstack.azurecr.io/postgres:${TAG}"
    environment:
      - "POSTGRES_PASSWORD=${DB_PASS}"
      - "PGDATA=/var/lib/postgresql/data/pgdata"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    networks:
      - utmstack-net
    command: ["postgres", "-c", "shared_buffers=256MB", "-c", "max_connections=1000"]

  frontend:
    container_name: frontend
    restart: always
    image: "utmstack.azurecr.io/utmstack_frontend:${TAG}"
    depends_on:
      - "panel"
      - "filebrowser"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ${CERT}:/etc/nginx/cert
    networks:
      - utmstack-net

  datasources_aws:
    container_name: datasources_aws
    restart: always
    image: "utmstack.azurecr.io/datasources:${TAG}"
    depends_on:
      - "node1"
      - "postgres"
    volumes:
      - ${UTMSTACK_DATASOURCES}:/etc/utmstack
    environment:
      - SERVER_NAME
      - DB_HOST
      - DB_PASS
    networks:
      - utmstack-net
    command: ["python3", "-m", "utmstack.aws"]

  datasources_office365:
    container_name: datasources_office365
    restart: always
    image: "utmstack.azurecr.io/datasources:${TAG}"
    depends_on:
      - "node1"
      - "postgres"
    volumes:
      - ${UTMSTACK_DATASOURCES}:/etc/utmstack
    environment:
      - SERVER_NAME
      - DB_HOST
      - DB_PASS
    networks:
      - utmstack-net
    command: ["python3", "-m", "utmstack.office365"]
  
  datasources_azure:
    container_name: datasources_azure
    restart: always
    image: "utmstack.azurecr.io/datasources:${TAG}"
    depends_on:
      - "node1"
      - "postgres"
    volumes:
      - ${UTMSTACK_DATASOURCES}:/etc/utmstack
    environment:
      - SERVER_NAME
      - DB_HOST
      - DB_PASS
    networks:
      - utmstack-net
    command: ["python3", "-m", "utmstack.azure"]

  datasources_webroot:
    container_name: datasources_webroot
    restart: always
    image: "utmstack.azurecr.io/datasources:${TAG}"
    depends_on:
      - "node1"
      - "postgres"
    volumes:
      - ${UTMSTACK_DATASOURCES}:/etc/utmstack
    environment:
      - SERVER_NAME
      - DB_HOST
      - DB_PASS
    networks:
      - utmstack-net
    command: ["python3", "-m", "utmstack.webroot"]

  datasources_sophos:
    container_name: datasources_sophos
    restart: always
    image: "utmstack.azurecr.io/datasources:${TAG}"
    depends_on:
      - "node1"
      - "postgres"
    volumes:
      - ${UTMSTACK_DATASOURCES}:/etc/utmstack
    environment:
      - SERVER_NAME
      - DB_HOST
      - DB_PASS
    networks:
      - utmstack-net
    command: ["python3", "-m", "utmstack.sophos"]

  datasources_logan:
    container_name: datasources_logan
    restart: always
    image: "utmstack.azurecr.io/datasources:${TAG}"
    depends_on:
      - "node1"
      - "postgres"
    volumes:
      - ${UTMSTACK_DATASOURCES}:/etc/utmstack
    environment:
      - SERVER_NAME
      - DB_HOST
      - DB_PASS
    ports:
      - "50051:50051"
    networks:
      - utmstack-net
    command: ["python3", "-m", "utmstack.logan"]

  panel:
    container_name: panel
    restart: always
    image: "utmstack.azurecr.io/utmstack_backend:${TAG}"
    depends_on:
      - "node1"
      - "postgres"
    environment:
      - SERVER_NAME
      - LITE
      - DB_USER=postgres
      - DB_PASS
      - DB_HOST
      - DB_PORT=5432
      - DB_NAME=utmstack
      - ELASTICSEARCH_HOST=${DB_HOST}
      - ELASTICSEARCH_PORT=9200
      - TOMCAT_ADMIN_USER=admin
      - "TOMCAT_ADMIN_PASSWORD=${DB_PASS}"
      - POSTGRESQL_USER=postgres
      - "POSTGRESQL_PASSWORD=${DB_PASS}"
      - POSTGRESQL_HOST=${DB_HOST}
      - POSTGRESQL_PORT=5432
      - POSTGRESQL_DATABASE=utmstack
      - OPENVAS_HOST=openvas
      - OPENVAS_PORT=9390
      - OPENVAS_USER=admin
      - "OPENVAS_PASSWORD=${DB_PASS}"
      - OPENVAS_PG_PORT=5432
      - OPENVAS_PG_DATABASE=gvmd
      - OPENVAS_PG_USER=gvm
      - "OPENVAS_PG_PASSWORD=${DB_PASS}"
      - JRE_HOME=/opt/tomcat/bin/jre
      - JAVA_HOME=/opt/tomcat/bin/jre
      - CATALINA_BASE=/opt/tomcat/
      - CATALINA_HOME=/opt/tomcat/
      - LD_LIBRARY_PATH=/usr/lib/x86_64-linux-gnu
    networks:
      - utmstack-net

  correlation:
    container_name: correlation
    restart: always
    image: "utmstack.azurecr.io/correlation:${TAG}"
    volumes:
      - geoip_data:/app/geosets
      - ${UTMSTACK_RULES}:/app/rulesets/custom
    ports:
      - "9090:8080"
    environment:
      - SERVER_NAME
      - POSTGRESQL_USER=postgres
      - "POSTGRESQL_PASSWORD=${DB_PASS}"
      - POSTGRESQL_HOST=${DB_HOST}
      - POSTGRESQL_PORT=5432
      - POSTGRESQL_DATABASE=utmstack
      - ELASTICSEARCH_HOST=${DB_HOST}
      - ELASTICSEARCH_PORT=9200
      - ERROR_LEVEL=info
    depends_on:
      - "node1"
      - "postgres"
    networks:
      - utmstack-net

  filebrowser:
    container_name: filebrowser
    restart: always
    image: "utmstack.azurecr.io/filebrowser:${TAG}"
    volumes:
      - ${UTMSTACK_RULES}:/srv
    environment:
      - "PASSWORD=${DB_PASS}"
    networks:
      - utmstack-net
`
	openvasTemplate = `
  openvas:
    container_name: openvas
    restart: always
    image: "utmstack.azurecr.io/openvas:${TAG}"
    volumes:
      - openvas_data:/data
    ports:
      - "8888:5432"
      - "9390:9390"
      - "9392:9392"
    environment:
      - USERNAME=admin
      - "PASSWORD=${DB_PASS}"
      - "DB_PASSWORD=${DB_PASS}"
      - HTTPS=0
    networks:
      - utmstack-net`

	probeTemplateStandard  = probeTemplateLite + openvasTemplate
	masterTemplateStandard = probeTemplateLite + masterTemplate + openvasTemplate
  masterTemplateLite = probeTemplateLite + masterTemplate
)
