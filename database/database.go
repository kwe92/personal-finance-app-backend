package database

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"personal_finance_backend/auth"

	"cloud.google.com/go/firestore"
)

type UserRecord struct {
	FirebaseUID      string `firestore:"firebaseUID"`
	Email            string `firestore:"email"`
	PlaidAccessToken string `firestore:"plaidAccessToken"`
	IsPlaidLinked    bool   `firestore:"is_plaid_linked"`
}

type Budget struct {
	ID        string    `firestore:"id,omitempty" json:"id"`
	Category  string    `firestore:"category" json:"category"`
	Maximum   float64   `firestore:"maximum" json:"maximum"`
	Theme     string    `firestore:"theme" json:"theme"`
	Period    string    `firestore:"period" json:"period"`
	StartDate string    `firestore:"startDate" json:"startDate"`
	EndDate   string    `firestore:"endDate" json:"endDate"`
	CreatedAt time.Time `firestore:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt" json:"updatedAt"`
}

type Store struct {
	firestoreClient *firestore.Client
}

var DefaultStore = &Store{}

func InitializeStore() {
	if DefaultStore.firestoreClient != nil {
		return
	}

	app, err := auth.GetFirebaseApp()
	if err != nil {
		log.Println("warning: unable to initialize Firebase app for Firestore persistence:", err)
		return
	}

	ctx := context.Background()
	client, err := app.Firestore(ctx)
	if err != nil {
		log.Println("warning: unable to initialize Firestore client:", err)
		return
	}

	DefaultStore.firestoreClient = client
}

func (s *Store) GetUser(firebaseUID string) (UserRecord, bool) {
	if s.firestoreClient == nil {
		return UserRecord{}, false
	}

	ctx := context.Background()
	doc, err := s.firestoreClient.Collection("users").Doc(firebaseUID).Get(ctx)
	if err != nil || !doc.Exists() {
		return UserRecord{}, false
	}

	var user UserRecord
	if err := doc.DataTo(&user); err != nil {
		log.Println("warning: Firestore GetUser decode failed:", err)
		return UserRecord{}, false
	}

	return user, true
}

func (s *Store) UpdatePlaidAccessToken(firebaseUID, accessToken string) error {
	if s.firestoreClient == nil {
		return errors.New("firestore client not initialized")
	}

	ctx := context.Background()
	_, err := s.firestoreClient.Collection("users").Doc(firebaseUID).Update(ctx, []firestore.Update{
		{Path: "plaidAccessToken", Value: accessToken},
		{Path: "is_plaid_linked", Value: true},
	})
	return err
}

// Calculate EndDate from StartDate + Period
func CalculateEndDate(startDateStr string, period string) string {
	parsedDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		parsedDate = time.Now()
		startDateStr = parsedDate.Format("2006-01-02")
	}

	var endDate time.Time
	switch strings.ToLower(period) {
	case "weekly":
		endDate = parsedDate.AddDate(0, 0, 6)
	case "biweekly":
		endDate = parsedDate.AddDate(0, 0, 13)
	case "monthly":
		endDate = parsedDate.AddDate(0, 1, 0).AddDate(0, 0, -1)
	default: // Default fallback to monthly
		endDate = parsedDate.AddDate(0, 1, 0).AddDate(0, 0, -1)
	}

	return endDate.Format("2006-01-02")
}

// Budget CRUD

func (s *Store) GetBudgets(firebaseUID string) ([]Budget, error) {
	if s.firestoreClient == nil {
		return nil, errors.New("firestore client not initialized")
	}

	ctx := context.Background()
	docs, err := s.firestoreClient.Collection("users").Doc(firebaseUID).Collection("budgets").Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	budgets := make([]Budget, 0, len(docs))
	for _, doc := range docs {
		var budget Budget
		if err := doc.DataTo(&budget); err != nil {
			log.Println("warning: Firestore GetBudgets decode failed:", err)
			continue
		}
		budget.ID = doc.Ref.ID
		budgets = append(budgets, budget)
	}

	return budgets, nil
}

func (s *Store) UpdateBudget(firebaseUID string, budgetID string, budget Budget) (Budget, error) {
	if s.firestoreClient == nil {
		return Budget{}, errors.New("firestore client not initialized")
	}

	if strings.TrimSpace(budget.Period) == "" {
		budget.Period = "monthly"
	}
	if strings.TrimSpace(budget.StartDate) == "" {
		budget.StartDate = time.Now().Format("2006-01-02")
	}

	budget.EndDate = CalculateEndDate(budget.StartDate, budget.Period)

	ctx := context.Background()
	budget.ID = budgetID
	budget.UpdatedAt = time.Now()

	_, err := s.firestoreClient.Collection("users").Doc(firebaseUID).Collection("budgets").Doc(budgetID).Update(ctx, []firestore.Update{
		{Path: "category", Value: budget.Category},
		{Path: "maximum", Value: budget.Maximum},
		{Path: "theme", Value: budget.Theme},
		{Path: "period", Value: budget.Period},
		{Path: "startDate", Value: budget.StartDate},
		{Path: "endDate", Value: budget.EndDate},
		{Path: "updatedAt", Value: budget.UpdatedAt},
	})
	if err != nil {
		return Budget{}, err
	}

	return budget, nil
}

func (s *Store) DeleteBudget(firebaseUID string, budgetID string) error {
	if s.firestoreClient == nil {
		return errors.New("firestore client not initialized")
	}

	ctx := context.Background()
	_, err := s.firestoreClient.Collection("users").Doc(firebaseUID).Collection("budgets").Doc(budgetID).Delete(ctx)
	return err
}

func (s *Store) CreateBudget(firebaseUID string, budget Budget) (Budget, error) {
	if s.firestoreClient == nil {
		return Budget{}, errors.New("firestore client not initialized")
	}

	if strings.TrimSpace(budget.Period) == "" {
		budget.Period = "monthly"
	}
	if strings.TrimSpace(budget.StartDate) == "" {
		budget.StartDate = time.Now().Format("2006-01-02")
	}

	budget.EndDate = CalculateEndDate(budget.StartDate, budget.Period)

	ctx := context.Background()
	now := time.Now()
	budget.CreatedAt = now
	budget.UpdatedAt = now

	docRef, _, err := s.firestoreClient.Collection("users").Doc(firebaseUID).Collection("budgets").Add(ctx, budget)
	if err != nil {
		return Budget{}, err
	}

	budget.ID = docRef.ID
	return budget, nil
}

// Pots CRUD

type Pot struct {
	ID        string    `firestore:"id,omitempty" json:"id"`
	Name      string    `firestore:"name" json:"name"`
	Target    float64   `firestore:"target" json:"target"`
	Total     float64   `firestore:"total" json:"total"`
	Theme     string    `firestore:"theme" json:"theme"`
	CreatedAt time.Time `firestore:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt,omitempty" json:"updatedAt"`
}

func (s *Store) GetPots(firebaseUID string) ([]Pot, error) {
	if s.firestoreClient == nil {
		return nil, errors.New("firestore client not initialized")
	}

	ctx := context.Background()
	docs, err := s.firestoreClient.Collection("users").Doc(firebaseUID).Collection("pots").Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	pots := make([]Pot, 0, len(docs))
	for _, doc := range docs {
		var pot Pot
		if err := doc.DataTo(&pot); err != nil {
			log.Println("warning: Firestore GetPots decode failed:", err)
			continue
		}
		pot.ID = doc.Ref.ID
		pots = append(pots, pot)
	}

	return pots, nil
}

func (s *Store) CreatePot(firebaseUID string, pot Pot) (Pot, error) {
	if s.firestoreClient == nil {
		return Pot{}, errors.New("firestore client not initialized")
	}

	ctx := context.Background()
	now := time.Now()
	pot.CreatedAt = now
	pot.UpdatedAt = now

	docRef, _, err := s.firestoreClient.Collection("users").Doc(firebaseUID).Collection("pots").Add(ctx, pot)
	if err != nil {
		return Pot{}, err
	}

	pot.ID = docRef.ID
	return pot, nil
}

func (s *Store) UpdatePot(firebaseUID string, potID string, pot Pot) (Pot, error) {
	if s.firestoreClient == nil {
		return Pot{}, errors.New("firestore client not initialized")
	}

	ctx := context.Background()
	pot.ID = potID
	pot.UpdatedAt = time.Now()

	_, err := s.firestoreClient.Collection("users").Doc(firebaseUID).Collection("pots").Doc(potID).Update(ctx, []firestore.Update{
		{Path: "name", Value: pot.Name},
		{Path: "target", Value: pot.Target},
		{Path: "total", Value: pot.Total},
		{Path: "theme", Value: pot.Theme},
		{Path: "updatedAt", Value: pot.UpdatedAt},
	})
	if err != nil {
		return Pot{}, err
	}

	return pot, nil
}

func (s *Store) DeletePot(firebaseUID string, potID string) error {
	if s.firestoreClient == nil {
		return errors.New("firestore client not initialized")
	}

	ctx := context.Background()
	docRef := s.firestoreClient.Collection("users").Doc(firebaseUID).Collection("pots").Doc(potID)

	// Fetch pot to ensure total balance is 0 before allowing deletion
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		return err
	}

	var pot Pot
	if err := docSnap.DataTo(&pot); err != nil {
		return err
	}

	if pot.Total > 0 {
		return errors.New("cannot delete pot with remaining funds; withdraw all funds first")
	}

	_, err = docRef.Delete(ctx)
	return err
}
