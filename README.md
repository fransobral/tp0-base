# TP0: Docker + Comunicaciones + Concurrencia

Este proyecto implementa un sistema distribuido básico utilizando contenedores Docker. La idea es demostrar la comunicación entre procesos y la concurrencia mediante un servidor "echo" (desarrollado en Python) y múltiples clientes (desarrollados en Go).


- **Servidor Echo (Python):**
  - Escucha en el puerto 12345.
  - Por cada conexión, recibe un mensaje, lo registra y lo devuelve al cliente (eco).
  - Se utiliza un loop para aceptar conexiones de manera continua.

- **Clientes (Go):**
  - Cada cliente se configura con un identificador único y parámetros de conexión (por ejemplo, dirección del servidor, cantidad de mensajes a enviar y período entre mensajes).
  - Por cada mensaje, el cliente crea una nueva conexión, envía el mensaje, espera el eco del servidor y, tras recibirlo, cierra la conexión.
  - Una vez enviados todos los mensajes, el cliente registra que terminó su ejecución.

## Ejercicio 1: Generación dinámica de Docker Compose

- **Objetivo:**
  - Permitir la generación dinámica del archivo `docker-compose-dev.yaml` con la cantidad de clientes deseada, sin necesidad de modificar manualmente el archivo.

- **Implementación:**
  - Desarrollé un script bash (`generar-compose.sh`) junto a un generador en Python (`mi-generador.py`) para crear el archivo de forma automática.
  - El script genera los servicios necesarios: un servicio para el servidor y un servicio para cada cliente (nombrados `client1`, `client2`, etc.) según el parámetro indicado.
  - Utilicé un bucle `for` en el generador para recorrer la cantidad de clientes a crear, añadiendo en cada iteración un servicio en el archivo YAML e incrementando el identificador del cliente.


## Ejercicio 2: Configuración Externa sin Rebuild

- **Objetivo:**
  - Permitir que los cambios en los archivos de configuración (`config.yaml` para el cliente y `config.ini` para el servidor) se apliquen sin necesidad de reconstruir las imágenes Docker.

- **Implementación:**
  - Puse los archivos de configuración en los contenedores usando volúmenes en el docker-compose-dev.yaml.
  - Aseguré que tanto el cliente como el servidor lean sus configuraciones desde los archivos montados (no desde la imagen).

- **Problema y solución:**
  - Inicialmente definí en el docker-compose la variable CLI_LOG_LEVEL=DEBUG para el cliente, lo que provocaba que el valor se fijara en "DEBUG" sin importar lo que pusiera en config.yaml.
  - Para solucionar esto, eliminé la línea de la variable de entorno, permitiendo que el cliente tome el valor de log_level directamente desde el archivo de configuración montado.

## Cómo ejecutar el proyecto

### 1. Generar el archivo Docker Compose

Desde la raíz del proyecto, ejecutá:

```bash
./generar-compose.sh docker-compose-dev.yaml 5
```

_Esto generará el archivo `docker-compose-dev.yaml` configurado para 5 clientes (client1, client2, …, client5)._

### 2. Construir las imágenes Docker

Utilizá el Makefile para construir las imágenes del servidor y del cliente:

```bash
make docker-image
```

### 3. Levantar el sistema

Arrancá el ambiente de contenedores con:

```bash
make docker-compose-up
```

### 4. Verificar la ejecución

Podés ver los logs para confirmar que los clientes se comunican correctamente con el servidor:

```bash
make docker-compose-logs
```

### 5. Detener el sistema

Para detener y eliminar los contenedores y recursos, ejecutá:

```bash
make docker-compose-down
```