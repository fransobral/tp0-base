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

## Ejercicio 6: Batch Processing de Apuestas

- **Objetivo:**
  - Adaptar los clientes para que envíen varias apuestas en una sola consulta (batch o chunk), reduciendo el tiempo de transmisión y el overhead de conexión.
  - La información de cada agencia se simula leyendo su archivo numérico correspondiente, siguiendo la convención de que el cliente N utilizará el archivo `.data/agency-{N}.csv`.
  - En el servidor, si todas las apuestas del batch se procesan correctamente, se debe loguear  
    `action: apuesta_recibida | result: success | cantidad: ${CANTIDAD_DE_APUESTAS}`; en caso contrario, se debe responder con un código de error y loguear  
    `action: apuesta_recibida | result: fail | cantidad: ${CANTIDAD_DE_APUESTAS}`.
  - La cantidad máxima de apuestas por batch es configurable mediante la clave `batch: maxAmount` en el archivo de configuración (`config.yaml`), y se ajustó para que los paquetes no excedan los 8kB.

- **Implementación:**
  - **Cliente:**  
    - Modifiqué el cliente para que, en lugar de enviar una apuesta a la vez, lea el archivo de apuestas (por ejemplo, `/app/.data/agency-<CLI_ID>.csv`) que se inyecta en el contenedor mediante volúmenes.
    - Se implementa una función que lee el archivo CSV y carga cada línea (cada apuesta) en un slice.
    - Estas apuestas se agrupan en batches (chunks) cuyo tamaño máximo es el definido en `batch.maxAmount` de `config.yaml`.
    - Por cada batch, se abre una conexión TCP, se envía el batch (cada apuesta separada por salto de línea) y se espera la respuesta del servidor.
    - Se maneja SIGTERM para garantizar un cierre graceful.
  
  - **Servidor:**  
    - El servidor recibe el batch de apuestas (un chunk de texto de hasta 8kB), lo separa en líneas y para cada línea valida que tenga 5 campos (por ejemplo: `first_name,last_name,document,birthdate,number`).
    - Si todas las líneas son válidas, las procesa (creando objetos `Bet` y llamando a `store_bets(...)`) y loguea  
      `action: apuesta_recibida | result: success | cantidad: ${cantidad}`; de lo contrario, responde con un error y loguea  
      `action: apuesta_recibida | result: fail | cantidad: ${cantidad}`.
    - El servidor responde con éxito solamente si **todas** las apuestas del batch fueron procesadas correctamente.

- **Comunicación:**
  - Utilizo TCP para la comunicación, manteniendo el protocolo de sockets.
  - El batch se envía como un bloque de texto plano: cada apuesta se coloca en una línea (usando el delimitador de coma `,` para separar campos) y se termina el batch con un salto de línea `\n`.
  - Esto permite que el servidor lea el mensaje completo (evitando fenómenos de "short read") y procese cada línea por separado.
  - Implemento el manejo de errores tanto en el envío como en la recepción y parseo de los datos.

- **Complicaciones y soluciones:**
  - **Problema 1:** Inicialmente, tuve dificultades para manejar el montaje de los archivos de apuestas. Los clientes no encontraban sus archivos porque la carpeta `.data` no estaba correctamente montada en el contenedor.
    - **Solución:** Modifiqué el `mi-generador.py` para incluir la carpeta `.data` como volumen en cada cliente y actualicé el código del cliente para leer desde la ruta `/app/.data/agency-{CLI_ID}.csv`.
  - **Problema 2:** Tuve complicaciones al agrupar las apuestas en batches, asegurándome de que cada paquete no excediera 8kB.
    - **Solución:** Implementé una función que divide el slice de apuestas en chunks del tamaño máximo configurado (definido en `batch.maxAmount`), lo que garantiza que cada mensaje enviado se mantenga por debajo del límite.

# Ejercicio 7 – Notificación y Consulta de Ganadores del Sorteo

## Objetivo

El objetivo de este ejercicio es ampliar la solución del ejercicio 6 para coordinar el sorteo de apuestas entre múltiples agencias. Cada cliente (agencia) envía sus apuestas en lotes y, al finalizar, notifica al servidor que ha concluido su envío. Una vez que todas las agencias han notificado, el servidor ejecuta el sorteo evaluando cada apuesta con las funciones provistas (`load_bets` y `has_won`) y almacena, para cada agencia, únicamente los DNI ganadores. Posteriormente, cada cliente consulta el resultado del sorteo, recibiendo solo la información correspondiente a su agencia.

## Implementación

### En el Servidor

- **Recepción y Procesamiento de Apuestas:**  
  El servidor recibe lotes de apuestas. Cada lote se separa en líneas, y para cada línea se valida que contenga los 5 campos requeridos. Si el formato es correcto, se crea un objeto `Bet` y se almacena mediante la función `store_bets`.

- **Notificación y Ejecución del Sorteo:**  
  Cuando un cliente termina de enviar sus apuestas, notifica al servidor con el mensaje `notify_finished|<agency_id>`.  
  - *Problema inicial:*  
    Mi primer enfoque usaba una variable `seen_agencies` para rastrear las agencias conectadas, lo que hacía que el sorteo se ejecutara tan pronto como se conectaba alguna agencia, sin esperar a que todas terminaran.
  - *Solución:*  
    Introduje el parámetro `expected_agencies`, obtenido de la variable de entorno `TOTAL_CLIENTES` (configurado en Docker Compose). El servidor ahora espera hasta que el número de agencias notificadas sea igual al total esperado antes de ejecutar el sorteo.

- **Consulta de Ganadores:**  
  Los clientes consultan los resultados mediante `query_winners|<agency_id>`.  
  - *Problema inicial:*  
    Cuando un cliente consultaba antes de que el sorteo estuviera completo, el servidor respondía con un error (`fail|sorteo_no_listo`), impidiendo reintentos.
  - *Solución:*  
    Se modificó la respuesta para que el servidor envíe `in_progress-sorteo_no_listo`, lo que permite al cliente reintentar la consulta cada 1 segundo (hasta un máximo de 30 reintentos) hasta obtener el resultado final.

### En el Cliente

- **Envío de Apuestas:**  
  Cada cliente lee su archivo CSV de apuestas (ubicado en `/app/.data/agency-<ID>.csv`), lo divide en lotes (chunks) con un tamaño máximo configurable y envía cada lote al servidor mediante conexiones TCP.

- **Notificación y Consulta:**  
  Una vez enviados todos los lotes, el cliente notifica al servidor con `notify_finished|<agency_id>` y, a continuación, consulta los ganadores. Si la respuesta es de "in_progress", el cliente reintenta la consulta, esperando que el sorteo se complete y se devuelvan los resultados pertinentes.

## Comunicación

- **Protocolo de Mensajes:**  
  La comunicación se realiza vía TCP. Los mensajes tienen el siguiente formato:
  - **Envío de apuestas:**  
    Un bloque de texto que comienza con `agency_ID|<ID>` seguido por cada apuesta en una línea separada (los campos de cada apuesta están separados por comas).
  - **Notificación de finalización:**  
    `notify_finished|<agency_id>`
  - **Consulta de ganadores:**  
    `query_winners|<agency_id>`

- **Respuestas del Servidor:**  
  - Durante la consulta de ganadores, si el sorteo aún no ha sido ejecutado, el servidor responde con `in_progress-sorteo_no_listo`, lo que permite al cliente reintentar.
  - Una vez que el sorteo se ha realizado, la respuesta es del formato `ok|<N>` seguido de cada DNI ganador en una línea separada.

## Complicaciones y Soluciones

- **Sincronización del Sorteo:**  
  - *Problema:* Usar `seen_agencies` para determinar cuándo iniciar el sorteo resultaba en una ejecución prematura, ya que se tomaba en cuenta tan pronto como alguna agencia se conectaba.
  - *Solución:* Se introdujo el parámetro `expected_agencies` (definido mediante la variable de entorno `TOTAL_CLIENTES` en Docker Compose), lo que permite que el servidor espere hasta recibir notificaciones de todas las agencias antes de realizar el sorteo.

- **Manejo de la Consulta de Ganadores Antes del Sorteo:**  
  - *Problema:* Si un cliente consultaba los ganadores antes de que el sorteo estuviera listo, el servidor respondía con `fail|sorteo_no_listo`, impidiendo reintentos y generando errores en los tests.
  - *Solución:* Se modificó la respuesta para devolver `in_progress-sorteo_no_listo`, permitiendo al cliente reintentar la consulta de forma periódica (cada 1 segundo) hasta que se complete el sorteo.

- **Reintentos en el Cliente:**  
  Se implementó en el cliente (en la función `QueryWinners` en `client.go`) un mecanismo de reintentos con un intervalo de 1 segundo entre cada intento y un máximo de 30 reintentos. Esto garantiza que el cliente logre obtener la respuesta una vez que el sorteo esté completo.

- **Distribución Específica de Resultados:**  
  El servidor almacena los resultados del sorteo en un diccionario (`_winners_by_agency`) de forma que cada agencia solo reciba los DNI ganadores correspondientes a ella, en lugar de hacer un broadcast global.

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
