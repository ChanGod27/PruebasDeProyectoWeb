package handlers

// =============================================================================
// AGREGADO PARA REDIS: HANDLERS Y HELPERS DE SESIÓN (redis_session_handler.go)
// =============================================================================
// Implementa la gestión de sesiones de usuario usando Redis como almacenamiento
// centralizado y una cookie HttpOnly opaca en el navegador.
// No modifica ningún handler existente para que pueda integrarse gradualmente.
// =============================================================================

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// NombreCookieSesion es el nombre de la cookie HttpOnly que guarda el token opaco en el navegador.
	NombreCookieSesion = "go_tasks_session"

	// DuracionSesion define cuánto tiempo permanece activa una sesión en Redis antes de expirar (24 horas).
	DuracionSesion = 24 * time.Hour

	// PrefijoClaveSesion es el prefijo usado en Redis para distinguir las sesiones de otros datos ("session:<token>").
	PrefijoClaveSesion = "session:"
)

// errRedisNoDisponible se devuelve cuando el cliente *redis.Client es nil (ej. Redis apagado al iniciar).
var errRedisNoDisponible = errors.New("cliente Redis no inicializado")

// estadoSesionResponse define la estructura JSON devuelta por SessionStatusRedisHandler.
type estadoSesionResponse struct {
	Autenticado bool   `json:"autenticado"`
	IDUsuario   int32  `json:"id_usuario"`
	Error       string `json:"error,omitempty"`
}

// CrearSesionRedis genera un token aleatorio criptográficamente seguro, lo guarda
// en Redis asociado al idUsuario con un TTL de 24 horas y envía la cookie HttpOnly al cliente.
func CrearSesionRedis(ctx context.Context, w http.ResponseWriter, rdb *redis.Client, idUsuario int32) error {
	// Validación defensiva para evitar panic si Redis no está conectado.
	if rdb == nil {
		return errRedisNoDisponible
	}

	// Genera 32 bytes aleatorios (256 bits) y los codifica en hexadecimal (64 caracteres).
	// Así nadie puede adivinar ni falsificar el identificador de sesión de otro usuario.
	bytesToken := make([]byte, 32)
	if _, err := rand.Read(bytesToken); err != nil {
		return err
	}
	token := hex.EncodeToString(bytesToken)

	// Guarda en Redis la relación "session:<token>" -> idUsuario con expiración automática (TTL).
	// Cuando pasen 24 horas sin actividad, Redis borrará la clave automáticamente.
	clave := PrefijoClaveSesion + token
	if err := rdb.Set(ctx, clave, idUsuario, DuracionSesion).Err(); err != nil {
		return err
	}

	// Envía únicamente el token aleatorio al navegador dentro de una cookie segura:
	// - HttpOnly: impide que JavaScript (document.cookie) pueda leer la sesión.
	// - SameSiteLaxMode: mitiga ataques CSRF manteniendo compatibilidad con enlaces y HTMX.
	// - Path "/": hace que la cookie se envíe en todas las rutas del sitio.
	http.SetCookie(w, &http.Cookie{
		Name:     NombreCookieSesion,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(DuracionSesion.Seconds()),
	})

	return nil
}

// ObtenerUsuarioIDRedis lee la cookie de sesión de la petición, consulta en Redis
// a qué usuario pertenece ese token y renueva el tiempo de expiración (sliding expiration).
func ObtenerUsuarioIDRedis(ctx context.Context, r *http.Request, rdb *redis.Client) (int32, error) {
	// Validación defensiva para evitar panic si Redis no está disponible.
	if rdb == nil {
		return 0, errRedisNoDisponible
	}

	// Busca la cookie "go_tasks_session" enviada automáticamente por el navegador.
	cookie, err := r.Cookie(NombreCookieSesion)
	if err != nil {
		return 0, err
	}
	if cookie.Value == "" {
		return 0, errors.New("cookie de sesión vacía")
	}

	// Consulta el ID del usuario almacenado en Redis bajo "session:<token>".
	// Si no existe o expiró, go-redis devuelve redis.Nil como error.
	clave := PrefijoClaveSesion + cookie.Value
	id, err := rdb.Get(ctx, clave).Int64()
	if err != nil {
		return 0, err
	}

	// Renueva el TTL en Redis en cada petición válida para mantener viva la sesión mientras el usuario navega.
	_ = rdb.Expire(ctx, clave, DuracionSesion).Err()

	return int32(id), nil
}

// CerrarSesionRedis elimina la clave de sesión en Redis (si existe la cookie y Redis está activo)
// e invalida la cookie en el navegador del usuario asignándole MaxAge: -1.
func CerrarSesionRedis(ctx context.Context, w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	// Si el navegador envió la cookie y Redis está disponible, borramos la sesión en el servidor
	// para que ese token no pueda volver a usarse aunque alguien lo haya copiado antes.
	if cookie, err := r.Cookie(NombreCookieSesion); err == nil && cookie.Value != "" && rdb != nil {
		_ = rdb.Del(ctx, PrefijoClaveSesion+cookie.Value).Err()
	}

	// Indica al navegador que elimine inmediatamente la cookie de sesión.
	http.SetCookie(w, &http.Cookie{
		Name:     NombreCookieSesion,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// RedirigirSinSesion es un helper para rutas protegidas:
//   - Si la petición proviene de HTMX (HX-Request: true), responde con HX-Redirect: /login y 401
//     para que HTMX redirija toda la página en lugar de inyectar el login dentro de un fragmento.
//   - Si es una petición HTTP normal, realiza una redirección estándar 303 See Other a /login.
func RedirigirSinSesion(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// LogoutRedisHandler atiende la ruta GET /logout:
// cierra la sesión en Redis, borra la cookie y redirige al usuario a /login
// soportando tanto navegación tradicional como peticiones HTMX.
func LogoutRedisHandler(w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	CerrarSesionRedis(r.Context(), w, r, rdb)

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// SessionStatusRedisHandler atiende la ruta GET /sesion/estado:
// verifica en Redis si hay una sesión activa y devuelve un JSON con "autenticado"
// e "id_usuario" para facilitar pruebas rápidas desde curl, Hurl o el navegador.
func SessionStatusRedisHandler(w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	w.Header().Set("Content-Type", "application/json")

	// Si Redis no fue inicializado, informamos el estado sin romper el servidor.
	if rdb == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(estadoSesionResponse{
			Autenticado: false,
			IDUsuario:   0,
			Error:       errRedisNoDisponible.Error(),
		})
		return
	}

	// Consulta si la cookie actual corresponde a una sesión válida en Redis.
	idUsuario, err := ObtenerUsuarioIDRedis(r.Context(), r, rdb)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(estadoSesionResponse{
			Autenticado: false,
			IDUsuario:   0,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(estadoSesionResponse{
		Autenticado: true,
		IDUsuario:   idUsuario,
	})
}
