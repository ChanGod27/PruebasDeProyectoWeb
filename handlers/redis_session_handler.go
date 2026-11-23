package handlers

// ==========================================
// OPERACIONES DE REDIS
// ==========================================

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	NombreCookieSesion = "go_tasks_session"

	DuracionSesion = 24 * time.Hour

	PrefijoClaveSesion = "session:"
)

var errRedisNoDisponible = errors.New("cliente Redis no inicializado")

// operacion principal de registro de la sesion y creacion de cookie
func CrearSesionRedis(ctx context.Context, w http.ResponseWriter, rdb *redis.Client, idUsuario int32) error {
	// Validación para evitar panic si Redis no está conectado.
	if rdb == nil {
		return errRedisNoDisponible
	}

	bytesToken := make([]byte, 32)
	if _, err := rand.Read(bytesToken); err != nil {
		return err
	}
	token := hex.EncodeToString(bytesToken)

	clave := PrefijoClaveSesion + token
	if err := rdb.Set(ctx, clave, idUsuario, DuracionSesion).Err(); err != nil {
		return err
	}

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

// se obtiene el usario que tiene sesion iniciada
func ObtenerUsuarioIDRedis(ctx context.Context, r *http.Request, rdb *redis.Client) (int32, error) {
	// Validación para evitar panic si Redis no está disponible.
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
	clave := PrefijoClaveSesion + cookie.Value
	id, err := rdb.Get(ctx, clave).Int64()
	if err != nil {
		return 0, err
	}

	// Renueva el TTL en Redis en cada petición válida para mantener viva la sesión mientras el usuario navega.
	_ = rdb.Expire(ctx, clave, DuracionSesion).Err()

	return int32(id), nil
}

// operacion de logout y eliminacion de cookie
func CerrarSesionRedis(ctx context.Context, w http.ResponseWriter, r *http.Request, rdb *redis.Client) {

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

// redirecciona a /login cuando  no hay sesion iniciada
func RedirigirSinSesion(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// cierra la sesión en Redis y redirige al usuario a /login
func LogoutRedisHandler(w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	CerrarSesionRedis(r.Context(), w, r, rdb)

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
