#!/usr/bin/env python3
import sys

def main():
    if len(sys.argv) != 3:
        print("La sintaxis es la siguiente: mi-generador.py <archivo_de_salida> <cantidad_de_clientes>")
        sys.exit(1)
    
    output_file = sys.argv[1]
    try:
        num_clientes = int(sys.argv[2])
    except ValueError:
        print("Error: La cantidad de clientes debe ser un número entero.")
        sys.exit(1)

    # Genero el contenido del docker-compose
    lines = []
    lines.append("name: tp0")
    lines.append("services:")
    lines.append("  server:")
    lines.append("    container_name: server")
    lines.append("    image: server:latest")
    lines.append("    entrypoint: python3 /main.py")
    lines.append("    environment:")
    lines.append("      - PYTHONUNBUFFERED=1")
    lines.append("      - LOGGING_LEVEL=DEBUG")
    lines.append("    networks:")
    lines.append("      - testing_net")
    lines.append("")

    # Genero los servicios para los clientes
    for i in range(1, num_clientes + 1):
        lines.append(f"  client{i}:")
        lines.append(f"    container_name: client{i}")
        lines.append("    image: client:latest")
        lines.append("    entrypoint: /client")
        lines.append("    environment:")
        lines.append(f"      - CLI_ID={i}")
        lines.append("      - CLI_LOG_LEVEL=DEBUG")
        lines.append("    networks:")
        lines.append("      - testing_net")
        lines.append("    depends_on:")
        lines.append("      - server")
        lines.append("")
    lines.append("networks:")
    lines.append("  testing_net:")
    lines.append("    ipam:")
    lines.append("      driver: default")
    lines.append("      config:")
    lines.append("        - subnet: 172.25.125.0/24")

    # Escribo el contenido en el archivo de salida
    with open(output_file, "w") as f:
        f.write("\n".join(lines))
    print(f"Archivo Docker Compose generado en '{output_file}' con {num_clientes} clientes.")

if __name__ == "__main__":
    main()
