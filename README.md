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

## Ej1

- **Generación dinámica de Docker Compose:**
  - Implementé un script bash (`generar-compose.sh`) junto a un generador en Python (`mi-generador.py`) para crear el archivo `docker-compose-dev.yaml` de forma dinámica.
  - El script genera los servicios necesarios: un servicio para el servidor y un servicio para cada cliente (nombrados `client1`, `client2`, etc.) según el parámetro indicado.
  - Para poder generar dinamicamente el archvio lo que hice fue crear un for que recorra la cantidad de clientes que se quieren generar y en cada iteración se agrega un servicio al archivo yaml aumentando el numero de id del cliente.

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