#!/bin/sh

if [ -f .env ]; then
  echo "=> Detectado arquivo .env dentro do container. Carregando variáveis..."
  export $(grep -v '^#' .env | xargs)
  echo "=> Variáveis do .env exportadas com sucesso!"
else
  echo "=> Nenhum arquivo .env detectado no container. Usando variáveis de ambiente globais."
fi

exec "$@"
