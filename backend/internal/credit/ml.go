package credit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// MLScorer implements the Scorer interface by calling an external ML service.
// This is the V3 upgrade path — deploy a Python ML model (e.g., XGBoost, LightGBM)
// behind a REST API and point this scorer at it.
//
// The ML service contract:
//
//	POST /predict
//	Request:  FeatureVector (JSON)
//	Response: MLPrediction (JSON)
//
// Example deployment:
//   - FastAPI + scikit-learn
//   - TensorFlow Serving
//   - Vertex AI / SageMaker endpoint
type MLScorer struct {
	endpoint   string        // e.g., "http://localhost:8092/predict"
	httpClient *http.Client
	version    string
}

// MLPrediction is the response from the ML service.
type MLPrediction struct {
	Score           int            `json:"score"`           // 300–850
	Confidence      float64        `json:"confidence"`      // 0.0–1.0
	FeatureWeights  map[string]float64 `json:"feature_weights"` // SHAP-like importance
	ModelVersion    string         `json:"model_version"`
}

// NewMLScorer creates a new ML-based scorer.
func NewMLScorer(endpoint string) *MLScorer {
	return &MLScorer{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		version: "ml-v3.0",
	}
}

func (s *MLScorer) Version() string { return s.version }

func (s *MLScorer) Score(ctx context.Context, fv *FeatureVector) (*ScoreResult, error) {
	body, err := json.Marshal(fv)
	if err != nil {
		return nil, fmt.Errorf("marshal features: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ml service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ml service returned status %d", resp.StatusCode)
	}

	var pred MLPrediction
	if err := json.NewDecoder(resp.Body).Decode(&pred); err != nil {
		return nil, fmt.Errorf("decode prediction: %w", err)
	}

	// Convert ML prediction to standard ScoreResult
	factors := make([]ScoreFactor, 0, len(pred.FeatureWeights))
	for name, weight := range pred.FeatureWeights {
		pts := int(weight * 100) // Normalize weight to points
		factors = append(factors, ScoreFactor{
			Category:    "ML_MODEL",
			Name:        name,
			Points:      pts,
			MaxPoints:   100,
			Percentage:  weight,
			Description: fmt.Sprintf("ML feature importance: %.2f", weight),
			Impact:      mlImpact(weight),
		})
	}

	return &ScoreResult{
		Score:        pred.Score,
		Grade:        ScoreGrade(pred.Score),
		Factors:      factors,
		ModelVersion: pred.ModelVersion,
		ComputedAt:   fv.ComputedAt,
		Features:     fv,
	}, nil
}

func mlImpact(weight float64) string {
	if weight > 0.1 {
		return "POSITIVE"
	}
	if weight > -0.05 {
		return "NEUTRAL"
	}
	return "NEGATIVE"
}

// HybridScorer ensembles rules + ML for a blended score.
// Use during the transition period while the ML model is being validated.
type HybridScorer struct {
	rules     *RulesScorer
	ml        *MLScorer
	mlWeight  float64 // 0.0 = pure rules, 1.0 = pure ML
}

// NewHybridScorer creates an ensemble scorer.
// mlWeight controls the blend: 0.3 means 30% ML + 70% rules.
func NewHybridScorer(rules *RulesScorer, ml *MLScorer, mlWeight float64) *HybridScorer {
	if mlWeight < 0 {
		mlWeight = 0
	}
	if mlWeight > 1 {
		mlWeight = 1
	}
	return &HybridScorer{
		rules:    rules,
		ml:       ml,
		mlWeight: mlWeight,
	}
}

func (s *HybridScorer) Version() string {
	return fmt.Sprintf("hybrid-v3.0(rules=%.0f%%,ml=%.0f%%)", (1-s.mlWeight)*100, s.mlWeight*100)
}

func (s *HybridScorer) Score(ctx context.Context, fv *FeatureVector) (*ScoreResult, error) {
	rulesResult, err := s.rules.Score(ctx, fv)
	if err != nil {
		return nil, fmt.Errorf("rules scorer: %w", err)
	}

	mlResult, err := s.ml.Score(ctx, fv)
	if err != nil {
		// ML service down — fall back to pure rules
		rulesResult.ModelVersion = "hybrid-v3.0(ml-fallback)"
		return rulesResult, nil
	}

	// Blend scores
	blendedScore := int(
		float64(rulesResult.Score)*(1-s.mlWeight) +
			float64(mlResult.Score)*s.mlWeight,
	)

	// Merge factors from both models
	allFactors := make([]ScoreFactor, 0, len(rulesResult.Factors)+len(mlResult.Factors))
	allFactors = append(allFactors, rulesResult.Factors...)
	allFactors = append(allFactors, mlResult.Factors...)

	// Merge suggestions
	allSuggestions := append(rulesResult.Suggestions, mlResult.Suggestions...)

	return &ScoreResult{
		Score:        blendedScore,
		Grade:        ScoreGrade(blendedScore),
		Factors:      allFactors,
		Suggestions:  allSuggestions,
		ModelVersion: s.Version(),
		ComputedAt:   fv.ComputedAt,
		Features:     fv,
	}, nil
}
