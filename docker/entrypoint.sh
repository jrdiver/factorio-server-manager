#!/bin/sh

init_config() {
    jq_cmd='.'

    if [ -n "$RCON_PASS" ]; then
      jq_cmd="${jq_cmd} | .rcon_pass = \"$RCON_PASS\""
      echo "Factorio rcon password is '$RCON_PASS'"
    fi

    if [ -n "$RCON_PORT" ]; then
      jq_cmd="${jq_cmd} | .rcon_port = $RCON_PORT"
      echo "Factorio rcon port is '$RCON_PORT'"
    fi

    jq_cmd="${jq_cmd} | .sq_lite_database_file = \"/opt/fsm-data/sqlite.db\""
    jq_cmd="${jq_cmd} | .log_file = \"/opt/fsm-data/factorio-server-manager.log\""

    if [ ! -f /opt/fsm-data/conf.json ]; then
      jq "${jq_cmd}" /opt/fsm/conf.json >/opt/fsm-data/conf.json
    else
      jq "${jq_cmd}" /opt/fsm-data/conf.json >/opt/fsm-data/conf.json.tmp && mv /opt/fsm-data/conf.json.tmp /opt/fsm-data/conf.json
    fi
}

random_pass() {
    LC_ALL=C tr -dc 'a-zA-Z0-9' </dev/urandom | fold -w 24 | head -n 1
}

install_game() {
    curl --location "https://www.factorio.com/get-download/${FACTORIO_VERSION}/headless/linux64" \
         --output /tmp/factorio_${FACTORIO_VERSION}.tar.xz
    tar -xf /tmp/factorio_${FACTORIO_VERSION}.tar.xz
    rm /tmp/factorio_${FACTORIO_VERSION}.tar.xz
}

init_config

install_game

rcon_port_arg=""
if [ -n "$RCON_PORT" ]; then
  rcon_port_arg="--rcon-port $RCON_PORT"
fi

rcon_pass_arg=""
if [ -n "$RCON_PASS" ]; then
  rcon_pass_arg="--rcon-password $RCON_PASS"
fi

cd /opt/fsm && ./factorio-server-manager --conf /opt/fsm-data/conf.json --dir /opt/factorio --port 80 $rcon_port_arg $rcon_pass_arg

