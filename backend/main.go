package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"my-app/database"
	"my-app/handlers"
	"my-app/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func absStorage() string {
	s := os.Getenv("STORAGE_ROOT")
	if s == "" {
		s = "./storage"
	}
	a, err := filepath.Abs(s)
	if err != nil {
		return s
	}
	return a
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: %v", err)
	}

	database.InitDB()
	database.RunMigrations()
	database.EnsureSchemaCompat()

	storageRoot := absStorage()
	if err := os.MkdirAll(storageRoot, 0o755); err != nil {
		log.Fatalf("storage: %v", err)
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) < 16 {
		log.Fatal("JWT_SECRET is required (min 16 characters). Set it in .env")
	}

	r := gin.Default()
	r.MaxMultipartMemory = 320 << 20

	authH := &handlers.AuthHandler{DB: database.DB, JWTSecret: jwtSecret, StorageRoot: storageRoot}
	adminH := &handlers.AdminHandler{DB: database.DB, JWTSecret: jwtSecret, StorageRoot: storageRoot}
	api := r.Group("/api")

	regRL := middleware.SlidingWindowRateLimit(8, time.Hour, middleware.ClientIPKey)
	loginRL := middleware.SlidingWindowRateLimit(25, 15*time.Minute, middleware.ClientIPKey)
	adminRL := middleware.SlidingWindowRateLimit(12, 15*time.Minute, middleware.ClientIPKey)
	landingRL := middleware.SlidingWindowRateLimit(40, time.Hour, middleware.ClientIPKey)

	api.POST("/auth/register", regRL, authH.Register)
	api.POST("/auth/login", loginRL, authH.Login)
	api.GET("/auth/verify", authH.VerifyEmail)
	api.POST("/admin/login", adminRL, adminH.Login)
	api.GET("/admin/overview", adminH.Overview)
	api.DELETE("/admin/workspaces/:id", adminH.DeleteWorkspace)
	api.DELETE("/admin/projects/:pid", adminH.DeleteProject)

	eh := &handlers.ExploreHandler{DB: database.DB}
	dh := &handlers.DatasetHandler{DB: database.DB, StorageRoot: storageRoot}
	api.GET("/public/landing-samples", landingRL, eh.LandingSamples)
	api.GET("/images/:imgid/file", middleware.OptionalJWTMiddleware(jwtSecret), dh.GetImageFile)

	authed := api.Group("")
	authed.Use(middleware.JWTMiddleware(jwtSecret))
	authed.GET("/me", authH.Me)
	authed.PATCH("/me", authH.UpdateProfile)
	authed.POST("/me/avatar", authH.UploadAvatar)
	authed.GET("/me/avatar", authH.Avatar)

	wh := &handlers.WorkspaceHandler{DB: database.DB, StorageRoot: storageRoot}
	authed.GET("/workspaces", wh.List)
	authed.POST("/workspaces", wh.Create)
	authed.DELETE("/workspaces/:id", wh.Delete)
	authed.GET("/workspaces/:id", wh.Get)
	authed.GET("/workspaces/:id/members", wh.ListMembers)
	authed.POST("/workspaces/:id/members", wh.AddMember)

	ph := &handlers.ProjectHandler{DB: database.DB, StorageRoot: storageRoot}
	authed.GET("/workspaces/:id/projects", ph.List)
	authed.POST("/workspaces/:id/projects", ph.Create)
	authed.GET("/projects/:pid", ph.Get)
	authed.PATCH("/projects/:pid", ph.Patch)
	authed.DELETE("/projects/:pid", ph.Delete)

	authed.GET("/explore/projects", eh.ListPublicProjects)
	authed.GET("/projects/:pid/versions", dh.ListVersions)
	authed.GET("/projects/:pid/versions/source-stats", dh.VersionSourceStats)
	authed.GET("/projects/:pid/versions/name-available", dh.VersionNameAvailable)
	authed.POST("/projects/:pid/versions/empty", dh.CreateEmptyVersion)
	authed.POST("/projects/:pid/versions/create-from-dataset", dh.CreateVersionFromDataset)
	authed.POST("/projects/:pid/uploads/images", dh.UploadProjectImages)
	authed.GET("/projects/:pid/batches", dh.ListProjectBatches)
	authed.GET("/projects/:pid/batches/:batch/start", dh.StartBatchAnnotating)
	authed.GET("/projects/:pid/batches/:batch/images", dh.ListBatchImages)
	authed.POST("/projects/:pid/batches/:batch/add-annotated", dh.AddBatchAnnotatedToDataset)
	authed.DELETE("/projects/:pid/batches/:batch", dh.DeleteBatch)
	authed.POST("/projects/:pid/versions/import-zip", dh.ImportZip)
	authed.POST("/projects/:pid/versions/import-folder", dh.ImportFolder)
	authed.GET("/projects/:pid/images", dh.ListProjectImages)
	authed.POST("/projects/:pid/images/delete", dh.DeleteProjectImages)
	authed.GET("/projects/:pid/class-stats", dh.ProjectClassStats)
	authed.GET("/import-jobs/:jid", dh.GetImportJob)
	authed.POST("/import-jobs/:jid/cancel", dh.CancelImportJob)
	authed.GET("/versions/:vid", dh.VersionMeta)
	authed.GET("/versions/:vid/class-stats", dh.VersionClassStats)
	authed.GET("/versions/:vid/split-stats", dh.VersionSplitStats)
	authed.PUT("/versions/:vid/data-yaml", dh.PutVersionDataYAML)
	authed.DELETE("/versions/:vid", dh.DeleteVersion)
	authed.GET("/versions/:vid/dataset.zip", dh.ExportVersionZIP)
	authed.PATCH("/versions/:vid/names", dh.PatchVersionNames)
	authed.GET("/versions/:vid/images", dh.ListImages)
	authed.GET("/images/:imgid/json", dh.GetImageJSON)
	authed.PUT("/images/:imgid/label", dh.PutLabel)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Alpha Guard API :%s - dataset images are available only on this machine: %s", port, storageRoot)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
