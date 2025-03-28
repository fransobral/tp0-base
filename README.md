# TP0: Docker + Comunicaciones + Concurrencia

En el presente repositorio se provee un esqueleto básico de cliente/servidor, en donde todas las dependencias del mismo se encuentran encapsuladas en containers. Los alumnos deberán resolver una guía de ejercicios incrementales, teniendo en cuenta las condiciones de entrega descritas al final de este enunciado.

 El cliente (Golang) y el servidor (Python) fueron desarrollados en diferentes lenguajes simplemente para mostrar cómo dos lenguajes de programación pueden convivir en el mismo proyecto con la ayuda de containers, en este caso utilizando [Docker Compose](https://docs.docker.com/compose/).

## Instrucciones de uso
El repositorio cuenta con un **Makefile** que incluye distintos comandos en forma de targets. Los targets se ejecutan mediante la invocación de:  **make \<target\>**. Los target imprescindibles para iniciar y detener el sistema son **docker-compose-up** y **docker-compose-down**, siendo los restantes targets de utilidad para el proceso de depuración.

Los targets disponibles son:

| target  | accion  |
|---|---|
|  `docker-compose-up`  | Inicializa el ambiente de desarrollo. Construye las imágenes del cliente y el servidor, inicializa los recursos a utilizar (volúmenes, redes, etc) e inicia los propios containers. |
| `docker-compose-down`  | Ejecuta `docker-compose stop` para detener los containers asociados al compose y luego  `docker-compose down` para destruir todos los recursos asociados al proyecto que fueron inicializados. Se recomienda ejecutar este comando al finalizar cada ejecución para evitar que el disco de la máquina host se llene de versiones de desarrollo y recursos sin liberar. |
|  `docker-compose-logs` | Permite ver los logs actuales del proyecto. Acompañar con `grep` para lograr ver mensajes de una aplicación específica dentro del compose. |
| `docker-image`  | Construye las imágenes a ser utilizadas tanto en el servidor como en el cliente. Este target es utilizado por **docker-compose-up**, por lo cual se lo puede utilizar para probar nuevos cambios en las imágenes antes de arrancar el proyecto. |
| `build` | Compila la aplicación cliente para ejecución en el _host_ en lugar de en Docker. De este modo la compilación es mucho más veloz, pero requiere contar con todo el entorno de Golang y Python instalados en la máquina _host_. |

### Servidor

Se trata de un "echo server", en donde los mensajes recibidos por el cliente se responden inmediatamente y sin alterar. 

Se ejecutan en bucle las siguientes etapas:

1. Servidor acepta una nueva conexión.
2. Servidor recibe mensaje del cliente y procede a responder el mismo.
3. Servidor desconecta al cliente.
4. Servidor retorna al paso 1.


### Cliente
 se conecta reiteradas veces al servidor y envía mensajes de la siguiente forma:
 
1. Cliente se conecta al servidor.
2. Cliente genera mensaje incremental.
3. Cliente envía mensaje al servidor y espera mensaje de respuesta.
4. Servidor responde al mensaje.
5. Servidor desconecta al cliente.
6. Cliente verifica si aún debe enviar un mensaje y si es así, vuelve al paso 2.

### Ejemplo

Al ejecutar el comando `make docker-compose-up`  y luego  `make docker-compose-logs`, se observan los siguientes logs:

```
client1  | 2024-08-21 22:11:15 INFO     action: config | result: success | client_id: 1 | server_address: server:12345 | loop_amount: 5 | loop_period: 5s | log_level: DEBUG
client1  | 2024-08-21 22:11:15 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°1
server   | 2024-08-21 22:11:14 DEBUG    action: config | result: success | port: 12345 | listen_backlog: 5 | logging_level: DEBUG
server   | 2024-08-21 22:11:14 INFO     action: accept_connections | result: in_progress
server   | 2024-08-21 22:11:15 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:15 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°1
server   | 2024-08-21 22:11:15 INFO     action: accept_connections | result: in_progress
server   | 2024-08-21 22:11:20 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:20 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°2
server   | 2024-08-21 22:11:20 INFO     action: accept_connections | result: in_progress
client1  | 2024-08-21 22:11:20 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°2
server   | 2024-08-21 22:11:25 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:25 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°3
client1  | 2024-08-21 22:11:25 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°3
server   | 2024-08-21 22:11:25 INFO     action: accept_connections | result: in_progress
server   | 2024-08-21 22:11:30 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:30 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°4
server   | 2024-08-21 22:11:30 INFO     action: accept_connections | result: in_progress
client1  | 2024-08-21 22:11:30 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°4
server   | 2024-08-21 22:11:35 INFO     action: accept_connections | result: success | ip: 172.25.125.3
server   | 2024-08-21 22:11:35 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°5
client1  | 2024-08-21 22:11:35 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°5
server   | 2024-08-21 22:11:35 INFO     action: accept_connections | result: in_progress
client1  | 2024-08-21 22:11:40 INFO     action: loop_finished | result: success | client_id: 1
client1 exited with code 0
```


## Parte 1: Introducción a Docker
En esta primera parte del trabajo práctico se plantean una serie de ejercicios que sirven para introducir las herramientas básicas de Docker que se utilizarán a lo largo de la materia. El entendimiento de las mismas será crucial para el desarrollo de los próximos TPs.

### Ejercicio N°1:
Definir un script de bash `generar-compose.sh` que permita crear una definición de Docker Compose con una cantidad configurable de clientes.  El nombre de los containers deberá seguir el formato propuesto: client1, client2, client3, etc. 

El script deberá ubicarse en la raíz del proyecto y recibirá por parámetro el nombre del archivo de salida y la cantidad de clientes esperados:

`./generar-compose.sh docker-compose-dev.yaml 5`

Considerar que en el contenido del script pueden invocar un subscript de Go o Python:

```
#!/bin/bash
echo "Nombre del archivo de salida: $1"
echo "Cantidad de clientes: $2"
python3 mi-generador.py $1 $2
```

En el archivo de Docker Compose de salida se pueden definir volúmenes, variables de entorno y redes con libertad, pero recordar actualizar este script cuando se modifiquen tales definiciones en los sucesivos ejercicios.

### Ejercicio N°2:
Modificar el cliente y el servidor para lograr que realizar cambios en el archivo de configuración no requiera reconstruír las imágenes de Docker para que los mismos sean efectivos. La configuración a través del archivo correspondiente (`config.ini` y `config.yaml`, dependiendo de la aplicación) debe ser inyectada en el container y persistida por fuera de la imagen (hint: `docker volumes`).


### Ejercicio N°3:
Crear un script de bash `validar-echo-server.sh` que permita verificar el correcto funcionamiento del servidor utilizando el comando `netcat` para interactuar con el mismo. Dado que el servidor es un echo server, se debe enviar un mensaje al servidor y esperar recibir el mismo mensaje enviado.

En caso de que la validación sea exitosa imprimir: `action: test_echo_server | result: success`, de lo contrario imprimir:`action: test_echo_server | result: fail`.

El script deberá ubicarse en la raíz del proyecto. Netcat no debe ser instalado en la máquina _host_ y no se pueden exponer puertos del servidor para realizar la comunicación (hint: `docker network`). `


### Ejercicio N°4:
Modificar servidor y cliente para que ambos sistemas terminen de forma _graceful_ al recibir la signal SIGTERM. Terminar la aplicación de forma _graceful_ implica que todos los _file descriptors_ (entre los que se encuentran archivos, sockets, threads y procesos) deben cerrarse correctamente antes que el thread de la aplicación principal muera. Loguear mensajes en el cierre de cada recurso (hint: Verificar que hace el flag `-t` utilizado en el comando `docker compose down`).

## Parte 2: Repaso de Comunicaciones

Las secciones de repaso del trabajo práctico plantean un caso de uso denominado **Lotería Nacional**. Para la resolución de las mismas deberá utilizarse como base el código fuente provisto en la primera parte, con las modificaciones agregadas en el ejercicio 4.

### Ejercicio N°5:
Modificar la lógica de negocio tanto de los clientes como del servidor para nuestro nuevo caso de uso.

#### Cliente
Emulará a una _agencia de quiniela_ que participa del proyecto. Existen 5 agencias. Deberán recibir como variables de entorno los campos que representan la apuesta de una persona: nombre, apellido, DNI, nacimiento, numero apostado (en adelante 'número'). Ej.: `NOMBRE=Santiago Lionel`, `APELLIDO=Lorca`, `DOCUMENTO=30904465`, `NACIMIENTO=1999-03-17` y `NUMERO=7574` respectivamente.

Los campos deben enviarse al servidor para dejar registro de la apuesta. Al recibir la confirmación del servidor se debe imprimir por log: `action: apuesta_enviada | result: success | dni: ${DNI} | numero: ${NUMERO}`.



#### Servidor
Emulará a la _central de Lotería Nacional_. Deberá recibir los campos de la cada apuesta desde los clientes y almacenar la información mediante la función `store_bet(...)` para control futuro de ganadores. La función `store_bet(...)` es provista por la cátedra y no podrá ser modificada por el alumno.
Al persistir se debe imprimir por log: `action: apuesta_almacenada | result: success | dni: ${DNI} | numero: ${NUMERO}`.

#### Comunicación:
Se deberá implementar un módulo de comunicación entre el cliente y el servidor donde se maneje el envío y la recepción de los paquetes, el cual se espera que contemple:
* Definición de un protocolo para el envío de los mensajes.
* Serialización de los datos.
* Correcta separación de responsabilidades entre modelo de dominio y capa de comunicación.
* Correcto empleo de sockets, incluyendo manejo de errores y evitando los fenómenos conocidos como [_short read y short write_](https://cs61.seas.harvard.edu/site/2018/FileDescriptors/).


### Ejercicio N°6:
Modificar los clientes para que envíen varias apuestas a la vez (modalidad conocida como procesamiento por _chunks_ o _batchs_). 
Los _batchs_ permiten que el cliente registre varias apuestas en una misma consulta, acortando tiempos de transmisión y procesamiento.

La información de cada agencia será simulada por la ingesta de su archivo numerado correspondiente, provisto por la cátedra dentro de `.data/datasets.zip`.
Los archivos deberán ser inyectados en los containers correspondientes y persistido por fuera de la imagen (hint: `docker volumes`), manteniendo la convencion de que el cliente N utilizara el archivo de apuestas `.data/agency-{N}.csv` .

En el servidor, si todas las apuestas del *batch* fueron procesadas correctamente, imprimir por log: `action: apuesta_recibida | result: success | cantidad: ${CANTIDAD_DE_APUESTAS}`. En caso de detectar un error con alguna de las apuestas, debe responder con un código de error a elección e imprimir: `action: apuesta_recibida | result: fail | cantidad: ${CANTIDAD_DE_APUESTAS}`.

La cantidad máxima de apuestas dentro de cada _batch_ debe ser configurable desde config.yaml. Respetar la clave `batch: maxAmount`, pero modificar el valor por defecto de modo tal que los paquetes no excedan los 8kB. 

Por su parte, el servidor deberá responder con éxito solamente si todas las apuestas del _batch_ fueron procesadas correctamente.

### Ejercicio N°7:

Modificar los clientes para que notifiquen al servidor al finalizar con el envío de todas las apuestas y así proceder con el sorteo.
Inmediatamente después de la notificacion, los clientes consultarán la lista de ganadores del sorteo correspondientes a su agencia.
Una vez el cliente obtenga los resultados, deberá imprimir por log: `action: consulta_ganadores | result: success | cant_ganadores: ${CANT}`.

El servidor deberá esperar la notificación de las 5 agencias para considerar que se realizó el sorteo e imprimir por log: `action: sorteo | result: success`.
Luego de este evento, podrá verificar cada apuesta con las funciones `load_bets(...)` y `has_won(...)` y retornar los DNI de los ganadores de la agencia en cuestión. Antes del sorteo no se podrán responder consultas por la lista de ganadores con información parcial.

Las funciones `load_bets(...)` y `has_won(...)` son provistas por la cátedra y no podrán ser modificadas por el alumno.

No es correcto realizar un broadcast de todos los ganadores hacia todas las agencias, se espera que se informen los DNIs ganadores que correspondan a cada una de ellas.

## Parte 3: Repaso de Concurrencia
En este ejercicio es importante considerar los mecanismos de sincronización a utilizar para el correcto funcionamiento de la persistencia.

### Ejercicio N°8:

Modificar el servidor para que permita aceptar conexiones y procesar mensajes en paralelo. En caso de que el alumno implemente el servidor en Python utilizando _multithreading_,  deberán tenerse en cuenta las [limitaciones propias del lenguaje](https://wiki.python.org/moin/GlobalInterpreterLock).

## Condiciones de Entrega
Se espera que los alumnos realicen un _fork_ del presente repositorio para el desarrollo de los ejercicios y que aprovechen el esqueleto provisto tanto (o tan poco) como consideren necesario.

Cada ejercicio deberá resolverse en una rama independiente con nombres siguiendo el formato `ej${Nro de ejercicio}`. Se permite agregar commits en cualquier órden, así como crear una rama a partir de otra, pero al momento de la entrega deberán existir 8 ramas llamadas: ej1, ej2, ..., ej7, ej8.
 (hint: verificar listado de ramas y últimos commits con `git ls-remote`)

Se espera que se redacte una sección del README en donde se indique cómo ejecutar cada ejercicio y se detallen los aspectos más importantes de la solución provista, como ser el protocolo de comunicación implementado (Parte 2) y los mecanismos de sincronización utilizados (Parte 3).

Se proveen [pruebas automáticas](https://github.com/7574-sistemas-distribuidos/tp0-tests) de caja negra. Se exige que la resolución de los ejercicios pase tales pruebas, o en su defecto que las discrepancias sean justificadas y discutidas con los docentes antes del día de la entrega. El incumplimiento de las pruebas es condición de desaprobación, pero su cumplimiento no es suficiente para la aprobación. Respetar las entradas de log planteadas en los ejercicios, pues son las que se chequean en cada uno de los tests.

La corrección personal tendrá en cuenta la calidad del código entregado y casos de error posibles, se manifiesten o no durante la ejecución del trabajo práctico. Se pide a los alumnos leer atentamente y **tener en cuenta** los criterios de corrección informados  [en el campus](https://campusgrado.fi.uba.ar/mod/page/view.php?id=73393).

---

# TP0: Docker + Comunicaciones + Concurrencia - Resolución

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
  Cuando un cliente termina de enviar sus apuestas, notifica al servidor con el mensaje `<longitud>;notify_finished|<agency_id>`.  
  - *Problema inicial:*  
    Mi primer enfoque usaba una variable `seen_agencies` para rastrear las agencias conectadas, lo que hacía que el sorteo se ejecutara tan pronto como se conectaba alguna agencia, sin esperar a que todas terminaran.
  - *Solución:*  
    Introduje el parámetro `expected_agencies`, obtenido de la variable de entorno `TOTAL_CLIENTES` (configurado en Docker Compose). El servidor ahora espera hasta que el número de agencias notificadas sea igual al total esperado antes de ejecutar el sorteo.

- **Consulta de Ganadores:**  
  Los clientes consultan los resultados mediante `<longitud>;query_winners|<agency_id>`.  
  - *Problema inicial:*  
    Cuando un cliente consultaba antes de que el sorteo estuviera completo, el servidor respondía con un error (`<longitud>;fail|sorteo_no_listo`), impidiendo reintentos.
  - *Solución:*  
    Se modificó la respuesta para que el servidor envíe `<longitud>;in_progress-sorteo_no_listo`, lo que permite al cliente reintentar la consulta cada 1 segundo (hasta un máximo de 30 reintentos) hasta obtener el resultado final.

### En el Cliente

- **Envío de Apuestas:**  
  Cada cliente lee su archivo CSV de apuestas (ubicado en `/app/.data/agency-<ID>.csv`), lo divide en lotes (chunks) con un tamaño máximo configurable y envía cada lote al servidor mediante conexiones TCP.

- **Notificación y Consulta:**  
  Una vez enviados todos los lotes, el cliente notifica al servidor con `<longitud>;notify_finished|<agency_id>` y, a continuación, consulta los ganadores. Si la respuesta es de "in_progress", el cliente reintenta la consulta, esperando que el sorteo se complete y se devuelvan los resultados pertinentes.

## Comunicación
- **Protocolo de Mensajes:**  
  La comunicación se realiza vía TCP. Los mensajes tienen el siguiente formato:
  - **Envío de apuestas:**  
    Un bloque de texto que comienza con `<longitud>;agency_ID|<ID>` seguido por cada apuesta en una línea separada (los campos de cada apuesta están separados por comas).
  - **Notificación de finalización:**  
    `<longitud>;notify_finished|<agency_id>`
  - **Consulta de ganadores:**  
    `<longitud>;query_winners|<agency_id>`

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

# Ejercicio 8 – Concurrencia en el Servidor

## Objetivo

- **Permitir el procesamiento concurrente:**  
  Modificar el servidor para que pueda aceptar conexiones y procesar mensajes en paralelo, atendiendo a múltiples clientes simultáneamente.
- **Consideración de la concurrencia en Python:**  
  Aunque CPython está limitado por el Global Interpreter Lock (GIL), la naturaleza I/O-bound de la aplicación permite que el uso de multithreading sea efectivo para este caso.

## Implementación

- **Procesamiento paralelo mediante multithreading:**  
  Modifiqué el bucle principal del servidor para que, al aceptar una conexión, se cree un nuevo hilo utilizando la librería `threading`. Cada hilo ejecuta la función `__handle_client_connection` de manera independiente.
- **Sincronización del estado compartido:**
Utilizo un threading.Lock para proteger las estructuras compartidas (como la lista de apuestas, el conjunto de agencias notificadas y el diccionario de ganadores). Esto garantiza que las actualizaciones sean atómicas y evita condiciones de carrera.

## Complicaciones y Soluciones
- **Concurrencia y estado compartido**  
  - *Problema:* Al procesar múltiples conexiones en paralelo, existía el riesgo de condiciones de carrera al modificar estructuras compartidas.
  - *Solución:* Implementé un Lock para asegurar que las actualizaciones del estado interno sean atómicas y seguras.
---
## Conclusión General

La evolución de este TP representó un camino progresivo desde una infraestructura básica hasta una solución distribuida robusta y concurrente, modelada en etapas:

### Evolución de la Solución

- **Ejercicio 1:** Comencé con la necesidad de escalar dinámicamente el entorno. Automatizar la generación de `docker-compose-dev.yaml` permitió trabajar con múltiples agencias sin modificar manualmente el archivo, y sentó las bases para simular un entorno distribuido.

- **Ejercicio 2:** Profundicé la separación entre código e infraestructura. Garanticé que los contenedores pudieran adaptarse a cambios de configuración sin rebuild, favoreciendo la agilidad en el desarrollo.

- **Ejercicio 3:** Validé la comunicación básica usando `netcat`, asegurando que la red interna de Docker funcionara como canal confiable entre servicios.

- **Ejercicio 4:** Introduje el manejo de señales (`SIGTERM`) para lograr terminaciones ordenadas. Esta capacidad fue fundamental en etapas posteriores donde múltiples hilos y procesos se ejecutan en paralelo.

- **Ejercicio 5:** Se dio el salto al dominio del negocio, reemplazando el echo server por el procesamiento de apuestas. Diseñe un protocolo simple de texto plano con delimitadores, y estructuré la lógica en torno a entidades como `Bet`.

- **Ejercicio 6:** Incorporé el procesamiento por lotes (batch), permitiendo que los clientes envíen múltiples apuestas en una sola conexión. Esto mejoró el rendimiento general y permitió trabajar con archivos CSV reales.

- **Ejercicio 7:** El sistema pasó a coordinar múltiples agencias. Implementé una notificación de finalización por cliente y un sorteo centralizado que espera a todas las agencias antes de responder. Esto introdujo lógica de sincronización y temporización de reintentos en el cliente.

- **Ejercicio 8:** Finalmente, adapté el servidor para manejar concurrencia real. Creé un hilo por conexión y protegí los datos compartidos con `threading.Lock`. Resolvi errores como `Broken pipe` mediante manejo robusto de excepciones.

### Protocolo de Comunicación

La solución utiliza un protocolo personalizado sobre TCP. Cada mensaje sigue un formato textual claro y bien definido:

- **Envío de apuestas:**  
  - Inicio con `<longitud>;agency_ID|<id>`  
  - Una línea por apuesta, campos separados por coma  
  - Respuesta del servidor: `success|N` o `fail|0`

- **Notificación de fin de envío:**  
  - Cliente: `<longitud>;notify_finished|<id>`  
  - Servidor: `ack_notify`

- **Consulta de ganadores:**  
  - Cliente: `<longitud>;query_winners|<id>`  
  - Servidor:
    - `in_progress-sorteo_no_listo` si el sorteo no se ejecutó aún
    - `ok|N` + N líneas con documentos ganadores si el sorteo está listo

Este diseño permitió una comunicación fluida entre clientes y servidor, con respuestas claras que habilitan reintentos seguros y controlados.

---

En resumen, la solución evolucionó desde una arquitectura monolítica con un único cliente, hasta una plataforma distribuida, concurrente, con configuración externa, procesamiento por lotes, y coordinación sincronizada entre múltiples nodos. Modelé una solución modular, extensible y tolerante a errores.

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


