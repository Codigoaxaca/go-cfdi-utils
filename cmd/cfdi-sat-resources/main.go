// Comando cfdi-sat-resources administra los recursos del SAT (XSLT y XSD)
// que usa go-cfdi-utils: los descarga, revisa cambios, muestra su estado y
// los borra. Sirve para cron, un deploy o un Dockerfile.
//
//	go install github.com/codigoaxaca/go-cfdi-utils/cmd/cfdi-sat-resources@latest
//
//	cfdi-sat-resources status
//	cfdi-sat-resources update            # descarga lo que falte y revisa cambios
//	cfdi-sat-resources update -force     # descarga todo de nuevo
//	cfdi-sat-resources prepare           # solo descarga lo que falte
//	cfdi-sat-resources clear             # borra los recursos guardados
//	cfdi-sat-resources export DESTINO    # copia la versión activa
//
// La opción -dir (antes del comando) indica el directorio; por omisión es el
// mismo que usa la librería (satresources.DefaultDir()).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/codigoaxaca/go-cfdi-utils/satresources"
)

const usage = `uso: cfdi-sat-resources [-dir DIR] [-timeout DURACIÓN] [-q] COMANDO

Comandos:
  status            muestra la versión activa y cuándo se revisó
  update [-force]   descarga lo que falte y revisa cambios con el SAT;
                    -force descarga todo de nuevo sin usar la caché
  prepare           descarga solo lo que falte, sin revisar cambios
  clear             borra los recursos guardados
  export DESTINO    copia la versión activa (manifest.json y files/) a DESTINO

Opciones:
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("cfdi-sat-resources", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", satresources.DefaultDir(), "directorio de los recursos")
	timeout := flags.Duration("timeout", 10*time.Minute, "tiempo máximo de la operación")
	quiet := flags.Bool("q", false, "no mostrar avisos")
	flags.Usage = func() {
		fmt.Fprint(stderr, usage)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() == 0 {
		flags.Usage()
		return 2
	}
	level := slog.LevelInfo
	if *quiet {
		level = slog.LevelError
	}
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level}))
	store := satresources.New(satresources.Config{Dir: *dir, Logger: logger})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	cmd, rest := flags.Arg(0), flags.Args()[1:]
	var err error
	switch cmd {
	case "status":
		if len(rest) != 0 {
			flags.Usage()
			return 2
		}
		fmt.Fprint(stdout, store.Status())
	case "update":
		sub := flag.NewFlagSet("update", flag.ContinueOnError)
		sub.SetOutput(stderr)
		force := sub.Bool("force", false, "descargar todo de nuevo sin usar la caché")
		if err := sub.Parse(rest); err != nil || sub.NArg() != 0 {
			flags.Usage()
			return 2
		}
		var opts []satresources.RefreshOption
		if *force {
			opts = append(opts, satresources.Force)
		}
		var res *satresources.RefreshResult
		if res, err = store.Refresh(ctx, opts...); err == nil {
			printResult(stdout, res)
		}
	case "prepare":
		if len(rest) != 0 {
			flags.Usage()
			return 2
		}
		if err = store.Prepare(ctx); err == nil {
			fmt.Fprintf(stdout, "recursos listos, versión %s\n", store.Version())
		}
	case "clear":
		if len(rest) != 0 {
			flags.Usage()
			return 2
		}
		if err = store.Clear(); err == nil {
			fmt.Fprintf(stdout, "recursos borrados de %s\n", store.Dir())
		}
	case "export":
		if len(rest) != 1 {
			flags.Usage()
			return 2
		}
		if err = store.Export(rest[0]); err == nil {
			fmt.Fprintf(stdout, "versión %s copiada a %s\n", store.Version(), rest[0])
		}
	default:
		fmt.Fprintf(stderr, "comando desconocido: %s\n\n", cmd)
		flags.Usage()
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		if errors.Is(err, satresources.ErrVerification) {
			return 3
		}
		return 1
	}
	return 0
}

func printResult(w io.Writer, res *satresources.RefreshResult) {
	if !res.Changed {
		fmt.Fprintf(w, "sin cambios, versión %s (%d archivos revisados)\n", res.Version, res.Checked)
		return
	}
	fmt.Fprintf(w, "versión nueva activada: %s (%d archivos)\n", res.Version, res.Checked)
	for _, group := range []struct {
		label string
		urls  []string
	}{{"actualizados", res.Updated}, {"nuevos", res.Added}, {"eliminados", res.Removed}} {
		if len(group.urls) > 0 {
			fmt.Fprintf(w, "%s:\n  %s\n", group.label, strings.Join(group.urls, "\n  "))
		}
	}
}
