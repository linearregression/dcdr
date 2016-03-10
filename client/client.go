package client

import (
	"hash/crc32"
	"strconv"

	"github.com/vsco/dcdr/cli/printer"
	"github.com/vsco/dcdr/client/models"
	"github.com/vsco/dcdr/client/watcher"
	"github.com/vsco/dcdr/config"
)

type ClientIFace interface {
	IsAvailable(feature string) bool
	IsAvailableForId(feature string, id uint64) bool
	ScaleValue(feature string, min float64, max float64) float64
	UpdateFeatures(bts []byte)
	FeatureExists(feature string) bool
	Features() models.Features
	FeatureMap() *models.FeatureMap
	ScopedMap() *models.FeatureMap
	Scopes() []string
	CurrentSha() string
	WithScopes(scopes ...string) *Client
}

type Client struct {
	featureMap *models.FeatureMap
	config     *config.Config
	watcher    watcher.WatcherIFace
	features   models.Features
	scopes     []string
}

func New(cfg *config.Config) (c *Client) {
	c = &Client{
		config: cfg,
	}

	if c.config.Watcher.OutputPath != "" {
		printer.Say("started watching %s", c.config.Watcher.OutputPath)
		c.watcher = watcher.NewWatcher(c.config.Watcher.OutputPath)
	}

	return
}

func NewDefault() (c *Client) {
	c = New(config.LoadConfig())

	return
}

func (c *Client) WithScopes(scopes ...string) *Client {
	if len(scopes) == 0 {
		return c
	}

	if len(scopes) == 1 && scopes[0] == "" {
		return c
	}

	newScopes := append(c.scopes, scopes...)

	newClient := &Client{
		featureMap: c.featureMap,
		scopes:     newScopes,
	}

	newClient.MergeScopes()

	if c.watcher != nil {
		newClient.watcher = watcher.NewWatcher(c.config.Watcher.OutputPath)
		newClient.Watch()
	}

	return newClient
}

func (c *Client) MergeScopes() {
	if c.featureMap != nil {
		c.features = c.featureMap.Dcdr.MergedScopes(c.scopes...)
	}
}

func (c *Client) Scopes() []string {
	return c.scopes
}

func (c *Client) SetFeatureMap(fm *models.FeatureMap) *Client {
	c.featureMap = fm

	c.MergeScopes()

	return c
}

func (c *Client) FeatureMap() *models.FeatureMap {
	return c.featureMap
}

func (c *Client) ScopedMap() *models.FeatureMap {
	fm := &models.FeatureMap{
		Dcdr: models.Root{
			Features: c.Features(),
		},
	}

	return fm
}

func (c *Client) Features() models.Features {
	return c.features
}

func (c *Client) CurrentSha() string {
	return c.FeatureMap().Dcdr.Info.CurrentSha
}

// FeatureExists checks the existence of a key
func (c *Client) FeatureExists(feature string) bool {
	_, exists := c.Features()[feature]

	return exists
}

// IsAvailable used to check features with boolean values.
func (c *Client) IsAvailable(feature string) bool {
	val, exists := c.Features()[feature]

	switch val.(type) {
	case bool:
		return exists && val.(bool)
	default:
		return false
	}
}

// IsAvailableForId used to check features with float values between 0.0-1.0.
func (c *Client) IsAvailableForId(feature string, id uint64) bool {
	val, exists := c.Features()[feature]

	switch val.(type) {
	case float64, int:
		return exists && c.withinPercentile(id, val.(float64), feature)
	default:
		return false
	}
}

func (c *Client) UpdateFeatures(bts []byte) {
	fm, err := models.NewFeatureMap(bts)

	if err != nil {
		printer.SayErr("parse error: %v", err)
		return
	}

	c.SetFeatureMap(fm)
}

// ScaleValue returns a value scaled between min and max
// given the current value of the feature.
func (c *Client) ScaleValue(feature string, min float64, max float64) float64 {
	val, exists := c.Features()[feature]

	if !exists {
		return min
	}

	switch val.(type) {
	case float64, int:
		return min + (max-min)*val.(float64)
	default:
		return min
	}
}

func (c *Client) Watch() (*Client, error) {
	if c.watcher != nil {
		err := c.watcher.Init()

		if err != nil {
			return nil, err
		}

		c.watcher.Register(c.UpdateFeatures)
		go c.watcher.Watch()
	}

	return c, nil
}

func (c *Client) withinPercentile(id uint64, val float64, feature string) bool {
	uid := c.crc(id, feature)
	percentage := uint32(val * 100)

	return uid%100 < percentage
}

func (c *Client) crc(id uint64, feature string) uint32 {
	b := []byte(feature + strconv.FormatInt(int64(id), 10))

	return crc32.ChecksumIEEE(b)
}