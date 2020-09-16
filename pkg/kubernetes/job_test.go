package kubernetes_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/billiford/go-clouddriver/pkg/kubernetes"
)

var _ = Describe("Job", func() {
	var (
		job Job
	)

	BeforeEach(func() {
		j := map[string]interface{}{}
		job = NewJob(j)
	})

	Describe("#State", func() {
		var s string

		JustBeforeEach(func() {
			s = job.State()
		})

		When("the job has not completed", func() {
			BeforeEach(func() {
				o := job.Object()
				o.Status.CompletionTime = nil
			})

			It("returns expected state", func() {
				Expect(s).To(Equal("Running"))
			})
		})

		When("the job has failed", func() {
			BeforeEach(func() {
				completions := int32(1)
				o := job.Object()
				o.Status.CompletionTime = &metav1.Time{}
				o.Spec.Completions = &completions
				o.Status.Conditions = []v1.JobCondition{
					{
						Type: "Failed",
					},
				}
			})

			It("returns expected state", func() {
				Expect(s).To(Equal("Failed"))
			})
		})

		When("the job is partially successful", func() {
			BeforeEach(func() {
				completions := int32(1)
				o := job.Object()
				o.Status.CompletionTime = &metav1.Time{}
				o.Spec.Completions = &completions
			})

			It("returns expected state", func() {
				Expect(s).To(Equal("Running"))
			})
		})

		When("the job succeeded", func() {
			BeforeEach(func() {
				completions := int32(1)
				o := job.Object()
				o.Status.CompletionTime = &metav1.Time{}
				o.Status.Succeeded = int32(1)
				o.Spec.Completions = &completions
			})

			It("returns expected state", func() {
				Expect(s).To(Equal("Succeeded"))
			})
		})
	})

	Describe("#Status", func() {
	})
})