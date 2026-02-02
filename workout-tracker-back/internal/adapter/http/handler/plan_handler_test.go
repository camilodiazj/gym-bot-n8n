package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gymbot/workout-tracker-back/internal/application/dto"
	"github.com/gymbot/workout-tracker-back/internal/application/usecase"
	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// MockPlanReader for testing
type MockPlanReader struct {
	GetMesocycleStatusFunc func(ctx context.Context, userID string) (*entity.MesocycleStatus, error)
	GetByUserIDFunc        func(ctx context.Context, userID string) (*entity.Plan, error)
}

func (m *MockPlanReader) GetMesocycleStatus(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
	if m.GetMesocycleStatusFunc != nil {
		return m.GetMesocycleStatusFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockPlanReader) GetByUserID(ctx context.Context, userID string) (*entity.Plan, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

// MockPlanRepository for testing
type MockPlanRepository struct {
	MockPlanReader
	IncrementMesocycleFunc   func(ctx context.Context, userID string) error
	UpdateWeekScheduleFunc   func(ctx context.Context, userID, weekSchedule string) error
}

func (m *MockPlanRepository) IncrementMesocycle(ctx context.Context, userID string) error {
	if m.IncrementMesocycleFunc != nil {
		return m.IncrementMesocycleFunc(ctx, userID)
	}
	return nil
}

func (m *MockPlanRepository) UpdateWeekSchedule(ctx context.Context, userID, weekSchedule string) error {
	if m.UpdateWeekScheduleFunc != nil {
		return m.UpdateWeekScheduleFunc(ctx, userID, weekSchedule)
	}
	return nil
}

// MockScheduleWriter for testing
type MockScheduleWriter struct {
	ClearScheduleFunc         func(ctx context.Context, userID string) error
	ClearScheduleForWeeksFunc func(ctx context.Context, userID string, weeks []int) error
}

func (m *MockScheduleWriter) ClearSchedule(ctx context.Context, userID string) error {
	if m.ClearScheduleFunc != nil {
		return m.ClearScheduleFunc(ctx, userID)
	}
	return nil
}

func (m *MockScheduleWriter) ClearScheduleForWeeks(ctx context.Context, userID string, weeks []int) error {
	if m.ClearScheduleForWeeksFunc != nil {
		return m.ClearScheduleForWeeksFunc(ctx, userID, weeks)
	}
	return nil
}

// MockWorkoutRenewalWriter for testing
type MockWorkoutRenewalWriter struct {
	DeleteUserWorkoutsFunc   func(ctx context.Context, userID string) error
	UpdateExerciseIDFunc     func(ctx context.Context, workoutID, newExerciseID string) error
	BatchUpdateExercisesFunc func(ctx context.Context, rotations []*entity.ExerciseRotation) error
	ResetWeeksToOneFunc      func(ctx context.Context, userID string) error
}

func (m *MockWorkoutRenewalWriter) DeleteUserWorkouts(ctx context.Context, userID string) error {
	if m.DeleteUserWorkoutsFunc != nil {
		return m.DeleteUserWorkoutsFunc(ctx, userID)
	}
	return nil
}

func (m *MockWorkoutRenewalWriter) UpdateExerciseID(ctx context.Context, workoutID, newExerciseID string) error {
	if m.UpdateExerciseIDFunc != nil {
		return m.UpdateExerciseIDFunc(ctx, workoutID, newExerciseID)
	}
	return nil
}

func (m *MockWorkoutRenewalWriter) BatchUpdateExercises(ctx context.Context, rotations []*entity.ExerciseRotation) error {
	if m.BatchUpdateExercisesFunc != nil {
		return m.BatchUpdateExercisesFunc(ctx, rotations)
	}
	return nil
}

func (m *MockWorkoutRenewalWriter) ResetWeeksToOne(ctx context.Context, userID string) error {
	if m.ResetWeeksToOneFunc != nil {
		return m.ResetWeeksToOneFunc(ctx, userID)
	}
	return nil
}

func TestNewPlanHandler(t *testing.T) {
	handler := NewPlanHandler(nil, nil, nil, nil, nil)
	if handler == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestPlanHandler_GetMesocycleStatus_Success(t *testing.T) {
	mockPlanRepo := &MockPlanReader{
		GetMesocycleStatusFunc: func(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
			return &entity.MesocycleStatus{
				UserID:          userID,
				MesocycleNumber: 1,
				DaysPerWeek:     4,
				WeekSchedule:    "ul_4",
				Week4Completed:  4,
				Week4Total:      4,
				IsComplete:      true,
				CompletionRate:  100.0,
				Goal:            "hipertrofia",
				Level:           "intermedio",
			}, nil
		},
	}

	checkStatusUC := usecase.NewCheckMesocycleStatusUseCase(mockPlanRepo)
	handler := NewPlanHandler(checkStatusUC, nil, nil, nil, nil)

	router := gin.New()
	router.GET("/plans/:userId/mesocycle-status", handler.GetMesocycleStatus)

	req, _ := http.NewRequest("GET", "/plans/user-123/mesocycle-status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Success bool                       `json:"success"`
		Data    dto.MesocycleStatusResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Success {
		t.Error("Expected success to be true")
	}
	if response.Data.MesocycleNumber != 1 {
		t.Errorf("Expected MesocycleNumber 1, got %d", response.Data.MesocycleNumber)
	}
	if !response.Data.IsComplete {
		t.Error("Expected IsComplete to be true")
	}
	if !response.Data.CanRenew {
		t.Error("Expected CanRenew to be true")
	}
}

func TestPlanHandler_GetMesocycleStatus_NotFound(t *testing.T) {
	mockPlanRepo := &MockPlanReader{
		GetMesocycleStatusFunc: func(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
			return nil, nil
		},
	}

	checkStatusUC := usecase.NewCheckMesocycleStatusUseCase(mockPlanRepo)
	handler := NewPlanHandler(checkStatusUC, nil, nil, nil, nil)

	router := gin.New()
	router.GET("/plans/:userId/mesocycle-status", handler.GetMesocycleStatus)

	req, _ := http.NewRequest("GET", "/plans/nonexistent-user/mesocycle-status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestPlanHandler_GetMesocycleStatus_NoUserID(t *testing.T) {
	handler := NewPlanHandler(nil, nil, nil, nil, nil)

	router := gin.New()
	// Intentionally not setting userId parameter
	router.GET("/plans/mesocycle-status", func(c *gin.Context) {
		c.Params = gin.Params{} // Empty params
		handler.GetMesocycleStatus(c)
	})

	req, _ := http.NewRequest("GET", "/plans/mesocycle-status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPlanHandler_RenewMaintain_Success(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		MockPlanReader: MockPlanReader{
			GetMesocycleStatusFunc: func(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
				return &entity.MesocycleStatus{
					UserID:          userID,
					MesocycleNumber: 1,
					DaysPerWeek:     4,
					WeekSchedule:    "ul_4",
					Week4Completed:  4,
					Week4Total:      4,
					IsComplete:      true,
				}, nil
			},
		},
		IncrementMesocycleFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}
	mockScheduleRepo := &MockScheduleWriter{
		ClearScheduleFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}
	mockWorkoutRepo := &MockWorkoutRenewalWriter{
		ResetWeeksToOneFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}

	renewMaintainUC := usecase.NewRenewMaintainUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)
	handler := NewPlanHandler(nil, renewMaintainUC, nil, nil, nil)

	router := gin.New()
	router.POST("/plans/:userId/renew/maintain", handler.RenewMaintain)

	req, _ := http.NewRequest("POST", "/plans/user-123/renew/maintain", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Success bool                 `json:"success"`
		Data    dto.RenewalResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Success {
		t.Error("Expected success to be true")
	}
	if response.Data.RenewalType != "MANTENER_RUTINA" {
		t.Errorf("Expected RenewalType MANTENER_RUTINA, got %s", response.Data.RenewalType)
	}
	if response.Data.NewMesocycleNumber != 2 {
		t.Errorf("Expected NewMesocycleNumber 2, got %d", response.Data.NewMesocycleNumber)
	}
}

func TestPlanHandler_RenewMaintain_NotComplete(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		MockPlanReader: MockPlanReader{
			GetMesocycleStatusFunc: func(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
				return &entity.MesocycleStatus{
					UserID:          userID,
					MesocycleNumber: 1,
					DaysPerWeek:     4,
					WeekSchedule:    "ul_4",
					Week4Completed:  2, // Only 2 of 4 completed
					Week4Total:      4,
					IsComplete:      false,
				}, nil
			},
		},
	}

	renewMaintainUC := usecase.NewRenewMaintainUseCase(mockPlanRepo, nil, nil)
	handler := NewPlanHandler(nil, renewMaintainUC, nil, nil, nil)

	router := gin.New()
	router.POST("/plans/:userId/renew/maintain", handler.RenewMaintain)

	req, _ := http.NewRequest("POST", "/plans/user-123/renew/maintain", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPlanHandler_RenewChangeDays_Success(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		MockPlanReader: MockPlanReader{
			GetMesocycleStatusFunc: func(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
				return &entity.MesocycleStatus{
					UserID:          userID,
					MesocycleNumber: 1,
					DaysPerWeek:     4,
					WeekSchedule:    "ul_4",
					Week4Completed:  4,
					Week4Total:      4,
					IsComplete:      true,
				}, nil
			},
		},
		IncrementMesocycleFunc: func(ctx context.Context, userID string) error {
			return nil
		},
		UpdateWeekScheduleFunc: func(ctx context.Context, userID, weekSchedule string) error {
			return nil
		},
	}
	mockScheduleRepo := &MockScheduleWriter{
		ClearScheduleFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}
	mockWorkoutRepo := &MockWorkoutRenewalWriter{
		DeleteUserWorkoutsFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}

	renewChangeDaysUC := usecase.NewRenewChangeDaysUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo)
	handler := NewPlanHandler(nil, nil, nil, renewChangeDaysUC, nil)

	router := gin.New()
	router.POST("/plans/:userId/renew/change-days", handler.RenewChangeDays)

	body := `{"new_days_per_week": 5}`
	req, _ := http.NewRequest("POST", "/plans/user-123/renew/change-days", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response struct {
		Success bool                   `json:"success"`
		Data    dto.ChangeDaysResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Success {
		t.Error("Expected success to be true")
	}
	if response.Data.RenewalType != "CAMBIAR_DIAS" {
		t.Errorf("Expected RenewalType CAMBIAR_DIAS, got %s", response.Data.RenewalType)
	}
	if response.Data.NewDaysPerWeek != 5 {
		t.Errorf("Expected NewDaysPerWeek 5, got %d", response.Data.NewDaysPerWeek)
	}
	if response.Data.OldDaysPerWeek != 4 {
		t.Errorf("Expected OldDaysPerWeek 4, got %d", response.Data.OldDaysPerWeek)
	}
	if !response.Data.RequiresRegeneration {
		t.Error("Expected RequiresRegeneration to be true when changing days")
	}
}

func TestPlanHandler_RenewChangeDays_InvalidDays(t *testing.T) {
	handler := NewPlanHandler(nil, nil, nil, nil, nil)

	router := gin.New()
	router.POST("/plans/:userId/renew/change-days", handler.RenewChangeDays)

	testCases := []struct {
		name string
		body string
	}{
		{"days too low", `{"new_days_per_week": 1}`},
		{"days too high", `{"new_days_per_week": 7}`},
		{"missing days", `{}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/plans/user-123/renew/change-days", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d for %s", w.Code, tc.name)
			}
		})
	}
}

func TestPlanHandler_RenewRotateExercises_NoUserID(t *testing.T) {
	handler := NewPlanHandler(nil, nil, nil, nil, nil)

	router := gin.New()
	router.POST("/plans/rotate", func(c *gin.Context) {
		c.Params = gin.Params{}
		handler.RenewRotateExercises(c)
	})

	req, _ := http.NewRequest("POST", "/plans/rotate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPlanHandler_RenewMaintain_NoUserID(t *testing.T) {
	handler := NewPlanHandler(nil, nil, nil, nil, nil)

	router := gin.New()
	router.POST("/plans/maintain", func(c *gin.Context) {
		c.Params = gin.Params{}
		handler.RenewMaintain(c)
	})

	req, _ := http.NewRequest("POST", "/plans/maintain", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPlanHandler_RenewChangeDays_NoUserID(t *testing.T) {
	handler := NewPlanHandler(nil, nil, nil, nil, nil)

	router := gin.New()
	router.POST("/plans/change-days", func(c *gin.Context) {
		c.Params = gin.Params{}
		handler.RenewChangeDays(c)
	})

	body := `{"new_days_per_week": 5}`
	req, _ := http.NewRequest("POST", "/plans/change-days", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPlanHandler_GetMesocycleStatus_InternalError(t *testing.T) {
	mockPlanRepo := &MockPlanReader{
		GetMesocycleStatusFunc: func(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
			return nil, apperror.NewInternalError("database error", nil)
		},
	}

	checkStatusUC := usecase.NewCheckMesocycleStatusUseCase(mockPlanRepo)
	handler := NewPlanHandler(checkStatusUC, nil, nil, nil, nil)

	router := gin.New()
	router.GET("/plans/:userId/mesocycle-status", handler.GetMesocycleStatus)

	req, _ := http.NewRequest("GET", "/plans/user-123/mesocycle-status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

// MockProfileRepository for testing
type MockProfileRepository struct {
	GetByUserIDFunc       func(ctx context.Context, userID string) (*entity.UserGymProfile, error)
	GetByWhatsAppIDFunc   func(ctx context.Context, whatsappID string) (*entity.UserGymProfile, error)
	UpdatePreferencesFunc func(ctx context.Context, whatsappID string, priorityMuscles, healthStatus, sessionDuration string) error
}

func (m *MockProfileRepository) GetByUserID(ctx context.Context, userID string) (*entity.UserGymProfile, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockProfileRepository) GetByWhatsAppID(ctx context.Context, whatsappID string) (*entity.UserGymProfile, error) {
	if m.GetByWhatsAppIDFunc != nil {
		return m.GetByWhatsAppIDFunc(ctx, whatsappID)
	}
	return nil, nil
}

func (m *MockProfileRepository) UpdatePreferences(ctx context.Context, whatsappID string, priorityMuscles, healthStatus, sessionDuration string) error {
	if m.UpdatePreferencesFunc != nil {
		return m.UpdatePreferencesFunc(ctx, whatsappID, priorityMuscles, healthStatus, sessionDuration)
	}
	return nil
}

func TestPlanHandler_RenewUpdateProfile_Success(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		MockPlanReader: MockPlanReader{
			GetMesocycleStatusFunc: func(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
				return &entity.MesocycleStatus{
					UserID:          userID,
					MesocycleNumber: 1,
					DaysPerWeek:     4,
					WeekSchedule:    "ul_4",
					Week4Completed:  4,
					Week4Total:      4,
					IsComplete:      true,
				}, nil
			},
		},
		IncrementMesocycleFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}
	mockScheduleRepo := &MockScheduleWriter{
		ClearScheduleFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}
	mockWorkoutRepo := &MockWorkoutRenewalWriter{
		DeleteUserWorkoutsFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}
	mockProfileRepo := &MockProfileRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.UserGymProfile, error) {
			return &entity.UserGymProfile{
				WhatsAppID: "573001234567@s.whatsapp.net",
				FullName:   "Test User",
			}, nil
		},
		UpdatePreferencesFunc: func(ctx context.Context, whatsappID string, priorityMuscles, healthStatus, sessionDuration string) error {
			return nil
		},
	}

	renewUpdateProfileUC := usecase.NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)
	handler := NewPlanHandler(nil, nil, nil, nil, renewUpdateProfileUC)

	router := gin.New()
	router.POST("/plans/:userId/renew/update-profile", handler.RenewUpdateProfile)

	body := `{"priority_muscles": "Gluteo, pierna", "health_status": "B"}`
	req, _ := http.NewRequest("POST", "/plans/user-123/renew/update-profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response struct {
		Success bool                       `json:"success"`
		Data    dto.UpdateProfileResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Success {
		t.Error("Expected success to be true")
	}
	if response.Data.NewMesocycleNumber != 2 {
		t.Errorf("Expected NewMesocycleNumber 2, got %d", response.Data.NewMesocycleNumber)
	}
	if !response.Data.RequiresRegeneration {
		t.Error("Expected RequiresRegeneration to be true")
	}
	if !response.Data.ProfileUpdated {
		t.Error("Expected ProfileUpdated to be true")
	}
	if !response.Data.WorkoutsDeleted {
		t.Error("Expected WorkoutsDeleted to be true")
	}
	if !response.Data.ScheduleCleared {
		t.Error("Expected ScheduleCleared to be true")
	}
}

func TestPlanHandler_RenewUpdateProfile_NoUserID(t *testing.T) {
	handler := NewPlanHandler(nil, nil, nil, nil, nil)

	router := gin.New()
	router.POST("/plans/update-profile", func(c *gin.Context) {
		c.Params = gin.Params{}
		handler.RenewUpdateProfile(c)
	})

	req, _ := http.NewRequest("POST", "/plans/update-profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPlanHandler_RenewUpdateProfile_MesocycleNotComplete(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		MockPlanReader: MockPlanReader{
			GetMesocycleStatusFunc: func(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
				return &entity.MesocycleStatus{
					UserID:          userID,
					MesocycleNumber: 1,
					DaysPerWeek:     4,
					WeekSchedule:    "ul_4",
					Week4Completed:  2, // Only 2 of 4 completed
					Week4Total:      4,
					IsComplete:      false,
				}, nil
			},
		},
	}
	mockProfileRepo := &MockProfileRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.UserGymProfile, error) {
			return &entity.UserGymProfile{
				WhatsAppID: "573001234567@s.whatsapp.net",
			}, nil
		},
	}

	renewUpdateProfileUC := usecase.NewRenewUpdateProfileUseCase(mockPlanRepo, nil, nil, mockProfileRepo)
	handler := NewPlanHandler(nil, nil, nil, nil, renewUpdateProfileUC)

	router := gin.New()
	router.POST("/plans/:userId/renew/update-profile", handler.RenewUpdateProfile)

	req, _ := http.NewRequest("POST", "/plans/user-123/renew/update-profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPlanHandler_RenewUpdateProfile_EmptyBody(t *testing.T) {
	mockPlanRepo := &MockPlanRepository{
		MockPlanReader: MockPlanReader{
			GetMesocycleStatusFunc: func(ctx context.Context, userID string) (*entity.MesocycleStatus, error) {
				return &entity.MesocycleStatus{
					UserID:          userID,
					MesocycleNumber: 1,
					DaysPerWeek:     4,
					WeekSchedule:    "ul_4",
					Week4Completed:  4,
					Week4Total:      4,
					IsComplete:      true,
				}, nil
			},
		},
		IncrementMesocycleFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}
	mockScheduleRepo := &MockScheduleWriter{
		ClearScheduleFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}
	mockWorkoutRepo := &MockWorkoutRenewalWriter{
		DeleteUserWorkoutsFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}
	mockProfileRepo := &MockProfileRepository{
		GetByUserIDFunc: func(ctx context.Context, userID string) (*entity.UserGymProfile, error) {
			return &entity.UserGymProfile{
				WhatsAppID: "573001234567@s.whatsapp.net",
			}, nil
		},
	}

	renewUpdateProfileUC := usecase.NewRenewUpdateProfileUseCase(mockPlanRepo, mockScheduleRepo, mockWorkoutRepo, mockProfileRepo)
	handler := NewPlanHandler(nil, nil, nil, nil, renewUpdateProfileUC)

	router := gin.New()
	router.POST("/plans/:userId/renew/update-profile", handler.RenewUpdateProfile)

	// Empty body - should still work, just no profile updates
	req, _ := http.NewRequest("POST", "/plans/user-123/renew/update-profile", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response struct {
		Success bool                       `json:"success"`
		Data    dto.UpdateProfileResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Success {
		t.Error("Expected success to be true")
	}
	if response.Data.ProfileUpdated {
		t.Error("Expected ProfileUpdated to be false when no updates provided")
	}
}
