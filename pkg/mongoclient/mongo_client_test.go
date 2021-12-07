package mongo

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ = Describe("MongoClient", func() {
	var (
		err           error
		localmongoURI = "mongodb://localhost:27017/"
		client        *mongo.Client
	)

	BeforeEach(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err = mongo.Connect(ctx, options.Client().ApplyURI(localmongoURI))
		Expect(err).To(BeNil())
	})

	AfterEach(func() {

	})

	Describe("Test FindOne", func() {
		It("should return the correct projected document", func() {
			filter := bson.D{}
			projection := bson.D{{"requestStatsInterval", 1}}

			var resultValue struct {
				RequestStatsInterval int32
			}

			err = FindOne(client, "globalSettings", filter, projection, &resultValue)
			Expect(err).To(BeNil())

			var expected int32 = 8
			Expect(resultValue.RequestStatsInterval).To(Equal(expected))
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Test UpdateOne", func() {
		It("should update the correct fields", func() {
			filter := bson.D{}
			projection := bson.D{{"hidden", 1}}

			type HiddenSettings struct {
				AutoEvictMissingDrive bool `bson:"autoEvictMissingDrive"`
				AutoFormatDrive       bool `bson:"autoFormatDrive"`
				IsElectDisabled       bool `bson:"isElectDisabled"`
			}

			type FindResult struct {
				ID     primitive.ObjectID `bson:"_id, omitempty"` // omitempty to protect against zeroed _id insertion
				Hidden HiddenSettings     `bson:"hidden"`
			}

			findResult := FindResult{}

			update := bson.D{{"$set", bson.D{
				{"hidden.autoEvictMissingDrive", true},
				{"hidden.autoFormatDrive", true}}}}

			err = UpdateOne(client, "globalSettings", filter, &update)
			Expect(err).To(BeNil())

			err = FindOne(client, "globalSettings", filter, projection, &findResult)
			Expect(err).To(BeNil())

			Expect(findResult.Hidden.AutoEvictMissingDrive).To(BeTrue())
			Expect(findResult.Hidden.AutoFormatDrive).To(BeTrue())

			update = bson.D{{"$set", bson.D{
				{"hidden.autoEvictMissingDrive", false},
				{"hidden.autoFormatDrive", false}}}}

			err = UpdateOne(client, "globalSettings", filter, &update)
			Expect(err).To(BeNil())

			err = FindOne(client, "globalSettings", filter, projection, &findResult)
			Expect(err).To(BeNil())

			Expect(findResult.Hidden.AutoEvictMissingDrive).To(BeFalse())
			Expect(findResult.Hidden.AutoFormatDrive).To(BeFalse())
		})
	})
})

func TestMongoClient(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "MongoClient Suite")
}
