package ee_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/sphericon/ee"
	"github.com/mokevnin/sphericon/ee/licensekey"
	"github.com/mokevnin/sphericon/ee/operator"
)

func licenseFor(t *testing.T, features ...licensekey.Feature) *licensekey.License {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	key, err := licensekey.Issue(priv, nil, features...)
	require.NoError(t, err)
	lic, err := licensekey.Parse(key, pub, time.Now())
	require.NoError(t, err)
	return lic
}

var siteSecret = rand.Text() + rand.Text()

func TestTheOperatorSecretIsRequiredOnlyWithTheOperatorLicense(t *testing.T) {
	unset := operator.Config{SessionTTL: time.Hour}

	e, err := ee.New(nil, licenseFor(t, licensekey.FeatureAudit), nil, siteSecret, unset)
	require.NoError(t, err, "an instance without the operator license needs no operator secret")
	assert.Nil(t, e.Operator(nil, nil), "and has no operator surface")

	_, err = ee.New(nil, licenseFor(t, licensekey.FeatureOperator), nil, siteSecret, unset)
	require.Error(t, err)
}

func TestTheOperatorSecretMustDifferFromTheSiteSecretAndBeLongEnough(t *testing.T) {
	lic := licenseFor(t, licensekey.FeatureOperator)

	_, err := ee.New(nil, lic, nil, siteSecret, operator.Config{Secret: siteSecret, SessionTTL: time.Hour})
	require.Error(t, err, "an Operator session must never verify as a User's")

	_, err = ee.New(nil, lic, nil, siteSecret, operator.Config{Secret: "short", SessionTTL: time.Hour})
	require.Error(t, err)

	e, err := ee.New(nil, lic, nil, siteSecret, operator.Config{Secret: rand.Text() + rand.Text(), SessionTTL: time.Hour})
	require.NoError(t, err)
	assert.NotNil(t, e.Operator(nil, nil))
}
