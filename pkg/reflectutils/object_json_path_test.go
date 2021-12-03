package reflectutils

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type Room struct {
	Size int32
	Name string
}

type Listing struct {
	Price       int32
	Description string
}

type Apartment struct {
	Size    int32
	Floor   int32
	Listing Listing
	Rooms   []Room
}

var _ = Describe("CompareObjectsAtJsonPath", func() {
	apt1 := Apartment{120, 3, Listing{500000, "apt"}, []Room{{25, "Living Room"}, {7, "Bathroom 1"}, {11, "Master Bedroom"}}}
	apt2 := Apartment{120, 2, Listing{400000, "apt"}, []Room{{25, "Living Room"}, {6, "Bathroom 1"}, {13, "Master Bedroom"}, {12, "Child Room 1"}}}

	Describe("Test first level", func() {
		It("should return true", func() {
			err, result := CompareObjectsAtJsonPath(apt1, apt2, "Size")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Equals).To(BeTrue())
		})

		It("should return false", func() {
			err, result := CompareObjectsAtJsonPath(apt1, apt2, "Floor")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Equals).To(BeFalse())
		})

		It("should return error", func() {
			err, result := CompareObjectsAtJsonPath(apt1, apt2, "NoField")
			Expect(err).To(HaveOccurred())
			Expect(result.Equals).To(BeFalse())
		})
	})

	Describe("Test second level", func() {
		It("should return true", func() {
			err, result := CompareObjectsAtJsonPath(apt1, apt2, "Listing.Description")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Equals).To(BeTrue())
		})

		It("should return false", func() {
			err, result := CompareObjectsAtJsonPath(apt1, apt2, "Listing.Price")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Equals).To(BeFalse())
		})

		It("should return error", func() {
			err, result := CompareObjectsAtJsonPath(apt1, apt2, "Listing.NoField")
			Expect(err).To(HaveOccurred())
			Expect(result.Equals).To(BeFalse())
		})
	})

	Describe("Test array items", func() {
		It("should return true", func() {
			err, result := CompareObjectsAtJsonPath(apt1, apt2, "Rooms[0].Size")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Equals).To(BeTrue())
		})

		It("should return false", func() {
			err, result := CompareObjectsAtJsonPath(apt1, apt2, "Rooms[1].Size")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Equals).To(BeFalse())
		})
	})
})

func TestCompareObjects(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "CompareObjects Suite")
}
