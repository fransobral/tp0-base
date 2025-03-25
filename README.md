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

## Ejercicio 3: Validación del Echo Server

- **Objetivo:**
  - Verificar el correcto funcionamiento del echo server sin exponer puertos en el host, utilizando netcat en un contenedor.

- **Implementación:**
  - Creé un script bash llamado `validar-echo-server.sh` ubicado en la raíz del proyecto.
  - El script lanza un contenedor basado en la imagen `busybox` (que incluye netcat) y lo une a la red interna de Docker (`tp0_testing_net`).
  - Dentro del contenedor, se usa netcat para enviar un mensaje de prueba ("Hello Echo") al servidor en el puerto 12345 y se captura la respuesta.
  - Se compara el mensaje enviado con la respuesta recibida:
    - Si son idénticos, se imprime:  
      `action: test_echo_server | result: success`
    - Si no coinciden, se imprime:  
      `action: test_echo_server | result: fail`

- **Notas:**
  - Utilizo busybox para evitar instalar netcat en el host.
  - La comunicación se realiza a través de la red interna de Docker, lo que permite validar el servicio sin exponer puertos al exterior.

## Ejercicio 4: Terminación Graceful con SIGTERM

- **Objetivo:**
  - Lograr que tanto el servidor como el cliente terminen de forma graceful al recibir la señal SIGTERM.
  - Asegurar que todos los recursos (sockets, archivos, threads, procesos) se cierren correctamente antes de que finalice la aplicación.
  - Registrar mensajes en el cierre de cada recurso para evidenciar que el shutdown se realizó de manera ordenada.

- **Implementación:**
  - En el servidor, modifiqué la clase para incluir una bandera interna `_running` y un método `shutdown()` que cierra el socket del servidor y detiene el loop principal.
  - En `main.py`, registré un handler para SIGTERM que, al recibir la señal, llama al método `shutdown()` del servidor y finaliza el proceso graceful.
  - En el cliente, incorporé un canal de señales para capturar SIGTERM en el bucle principal. Si se detecta la señal, el cliente registra el mensaje de salida y termina el bucle sin seguir enviando mensajes.
  - Añadí logs en cada paso del cierre (cierre de conexión, cierre del socket, etc.) para poder verificar en los registros que todos los recursos se liberaron correctamente.

## Ejercicio 5: Apuestas de la Lotería Nacional

- **Objetivo:**
  - Adaptar el cliente y el servidor para implementar un nuevo caso de uso: la recepción, procesamiento y registro de apuestas de distintas agencias de quiniela.
  - Reemplazar la lógica del echo server por una funcionalidad de negocio real, manteniendo el manejo de SIGTERM y la arquitectura modular previa.

- **Implementación:**
  - Modifiqué el cliente para que lea los datos de la apuesta (nombre, apellido, documento, nacimiento, número) desde variables de entorno.
  - Cada contenedor cliente representa una agencia distinta y envía su apuesta al servidor al arrancar.
  - El cliente construye un mensaje con los campos separados por `|` (sin usar librerías de serialización), lo envía al servidor y espera una confirmación.
  - El servidor recibe el mensaje, lo parsea, crea una instancia de `Bet` y lo almacena con `store_bets(...)`.
  - Se loguea `action: apuesta_almacenada` en el servidor y `action: apuesta_enviada` en el cliente si todo sale bien.
  - Conservo el manejo de SIGTERM en cliente y servidor para que ambos finalicen de manera graceful al recibir esa señal.

- **Comunicación:**
  - Implementé un protocolo propio donde cada mensaje de apuesta se envía como texto plano con campos separados por el caracter `|` y termina en `\n`.
  - No utilicé ninguna librería externa de serialización como JSON, cumpliendo con los requerimientos de la cátedra.
  - La separación de responsabilidades se mantiene: el cliente arma el mensaje desde datos de entorno, y el servidor lo desarma y lo convierte en un objeto de dominio (`Bet`).
  - Para evitar short reads, los mensajes incluyen un delimitador (`\n`) que permite leer el mensaje completo en el servidor.
  - Manejo los errores de envío, recepción y parseo de manera explícita, con logs claros en ambos extremos.

- **Protocolo de transporte:**
  - El sistema utiliza **TCP** tanto en cliente como en servidor:
    - `net.Dial("tcp", ...)` en el cliente.
    - `socket.SOCK_STREAM` en el servidor.
  - Esto garantiza entrega en orden y manejo automático de retransmisiones.

- **Control de flujo:**
  - Cada cliente implementa el esquema **"envío → espera confirmación → fin"**.
  - Este mecanismo evita que se saturen los buffers de TCP si el servidor no puede procesar rápidamente.
  - La conexión se cierra después de la confirmación, garantizando un envío seguro por vez.

- **Complicaciones y soluciones:**

  - **Problema 1:** Inicialmente utilizaba el delimitador `|` para separar los campos del mensaje. Sin embargo, al loguear el mensaje completo se rompía el parser de logs, ya que el contenido podía contener múltiples `|` y generar detalles vacíos o mal formateados (por ejemplo, obteniendo entradas como `"Sobr"` sin el separador esperado).
  - **Solución:** Decidí cambiar el delimitador a `~`, un carácter poco común en datos normales. Actualicé la función de serialización en el cliente para concatenar los campos usando `~`, y en el servidor modifiqué el parseo para usar este mismo delimitador. Con esto evito conflictos en los logs y mantuve la integridad del mensaje enviado.

  - **Problema 2:** Al correr los tests, me arrojaba error ya que no estaban inicializadas las variables de entorno `NOMBRE`, `APELLIDO`, `DOCUMENTO`, `NACIMIENTO` y `NUMERO`.
  - **Solución:** Agregué dichas variables a `mi-generador.py` para que se inicialicen con valores por defecto.


---

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
