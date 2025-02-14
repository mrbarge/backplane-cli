package script

import (
	"errors"
	"net/http"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	bpclient "github.com/openshift/backplane-api/pkg/client"
	"github.com/openshift/backplane-cli/pkg/client/mocks"
	"github.com/openshift/backplane-cli/pkg/info"
	"github.com/openshift/backplane-cli/pkg/utils"
	mocks2 "github.com/openshift/backplane-cli/pkg/utils/mocks"
)

var _ = Describe("describe script command", func() {

	var (
		mockCtrl         *gomock.Controller
		mockClient       *mocks.MockClientInterface
		mockOcmInterface *mocks2.MockOCMInterface
		mockClientUtil   *mocks2.MockClientUtils

		testClusterId  string
		testToken      string
		trueClusterId  string
		testScriptName string

		proxyUri string

		fakeResp *http.Response

		sut *cobra.Command
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClientInterface(mockCtrl)

		mockOcmInterface = mocks2.NewMockOCMInterface(mockCtrl)
		utils.DefaultOCMInterface = mockOcmInterface

		mockClientUtil = mocks2.NewMockClientUtils(mockCtrl)
		utils.DefaultClientUtils = mockClientUtil

		testClusterId = "test123"
		testToken = "hello123"
		trueClusterId = "trueID123"
		testScriptName = "CEE/abc"

		proxyUri = "https://shard.apps"

		sut = NewScriptCmd()

		fakeResp = &http.Response{
			Body: MakeIoReader(`
[
{
  "allowedGroups":["CEE"],
  "author":"author",
  "canonicalName":"CEE/abc",
  "description":"desc",
  "language":"Python",
  "path":"something",
  "permalink":"https://link",
  "rbac": {},
  "envs": [{"key":"var1","description":"variable 1","optional":false}]
}
]
`),
			Header:     map[string][]string{},
			StatusCode: http.StatusOK,
		}
		fakeResp.Header.Add("Content-Type", "json")
		// Clear config file
		_ = clientcmd.ModifyConfig(clientcmd.NewDefaultPathOptions(), api.Config{}, true)

		os.Setenv(info.BACKPLANE_URL_ENV_NAME, proxyUri)
	})

	AfterEach(func() {
		os.Setenv(info.BACKPLANE_URL_ENV_NAME, "")
		mockCtrl.Finish()
	})

	Context("describe script", func() {
		It("when running with a simple case should work as expected", func() {
			// It should query for the internal cluster id first
			mockOcmInterface.EXPECT().GetTargetCluster(testClusterId).Return(trueClusterId, testClusterId, nil)
			// Then it will look for the backplane shard
			mockOcmInterface.EXPECT().IsClusterHibernating(gomock.Eq(trueClusterId)).Return(false, nil).AnyTimes()
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(&testToken, nil).AnyTimes()
			mockClientUtil.EXPECT().MakeRawBackplaneAPIClient(gomock.Any()).Return(mockClient, nil)
			mockOcmInterface.EXPECT().GetTargetCluster(testClusterId).Return(trueClusterId, testClusterId, nil)
			mockClient.EXPECT().GetScriptsByCluster(gomock.Any(), trueClusterId, &bpclient.GetScriptsByClusterParams{Scriptname: &testScriptName}).Return(fakeResp, nil)

			sut.SetArgs([]string{"describe", testScriptName, "--cluster-id", testClusterId})
			err := sut.Execute()

			Expect(err).To(BeNil())
		})

		It("should respect url flag", func() {
			mockOcmInterface.EXPECT().IsClusterHibernating(gomock.Eq(trueClusterId)).Return(false, nil).AnyTimes()
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(&testToken, nil).AnyTimes()
			mockClientUtil.EXPECT().MakeRawBackplaneAPIClient("https://newbackplane.url").Return(mockClient, nil)
			mockOcmInterface.EXPECT().GetTargetCluster(testClusterId).Return(trueClusterId, testClusterId, nil)
			mockClient.EXPECT().GetScriptsByCluster(gomock.Any(), trueClusterId, &bpclient.GetScriptsByClusterParams{Scriptname: &testScriptName}).Return(fakeResp, nil)

			sut.SetArgs([]string{"describe", testScriptName, "--cluster-id", testClusterId, "--url", "https://newbackplane.url"})
			err := sut.Execute()

			Expect(err).To(BeNil())
		})

		It("Should able use the current logged in cluster if non specified and retrieve from config file", func() {
			err := utils.CreateTempKubeConfig(nil)
			Expect(err).To(BeNil())
			mockOcmInterface.EXPECT().IsClusterHibernating(gomock.Eq("configcluster")).Return(false, nil).AnyTimes()
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(&testToken, nil).AnyTimes()
			mockClientUtil.EXPECT().MakeRawBackplaneAPIClient("https://api-backplane.apps.something.com").Return(mockClient, nil)
			mockOcmInterface.EXPECT().GetTargetCluster("configcluster").Return(trueClusterId, testClusterId, nil)
			mockClient.EXPECT().GetScriptsByCluster(gomock.Any(), trueClusterId, &bpclient.GetScriptsByClusterParams{Scriptname: &testScriptName}).Return(fakeResp, nil)

			sut.SetArgs([]string{"describe", testScriptName})
			err = sut.Execute()

			Expect(err).To(BeNil())
		})

		It("should fail when backplane did not return a 200", func() {
			mockOcmInterface.EXPECT().GetTargetCluster(testClusterId).Return(trueClusterId, testClusterId, nil)
			mockOcmInterface.EXPECT().IsClusterHibernating(gomock.Eq(trueClusterId)).Return(false, nil).AnyTimes()
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(&testToken, nil).AnyTimes()
			mockClientUtil.EXPECT().MakeRawBackplaneAPIClient(gomock.Any()).Return(mockClient, nil)
			mockOcmInterface.EXPECT().GetTargetCluster(testClusterId).Return(trueClusterId, testClusterId, nil)
			mockClient.EXPECT().GetScriptsByCluster(gomock.Any(), trueClusterId, &bpclient.GetScriptsByClusterParams{Scriptname: &testScriptName}).Return(nil, errors.New("err"))

			sut.SetArgs([]string{"describe", testScriptName, "--cluster-id", testClusterId})
			err := sut.Execute()

			Expect(err).ToNot(BeNil())
		})

		It("should fail when backplane returns a blank script list with 200 status", func() {
			fakeRespBlank := &http.Response{
				Body:       MakeIoReader(`[]`),
				Header:     map[string][]string{},
				StatusCode: http.StatusOK,
			}
			fakeRespBlank.Header.Add("Content-Type", "json")

			mockOcmInterface.EXPECT().GetTargetCluster(testClusterId).Return(trueClusterId, testClusterId, nil)
			mockOcmInterface.EXPECT().IsClusterHibernating(gomock.Eq(trueClusterId)).Return(false, nil).AnyTimes()
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(&testToken, nil).AnyTimes()
			mockClientUtil.EXPECT().MakeRawBackplaneAPIClient(gomock.Any()).Return(mockClient, nil)
			mockOcmInterface.EXPECT().GetTargetCluster(testClusterId).Return(trueClusterId, testClusterId, nil)
			mockClient.EXPECT().GetScriptsByCluster(gomock.Any(), trueClusterId, &bpclient.GetScriptsByClusterParams{Scriptname: &testScriptName}).Return(fakeRespBlank, nil)

			sut.SetArgs([]string{"describe", testScriptName, "--cluster-id", testClusterId})
			err := sut.Execute()

			Expect(err).ToNot(BeNil())
		})

		It("should still work if the result does not contain envs", func() {
			fakeRespNoEnv := &http.Response{
				Body: MakeIoReader(`[
{
  "allowedGroups":["CEE"],
  "author":"author",
  "canonicalName":"CEE/abc",
  "description":"desc",
  "language":"Python",
  "path":"something",
  "permalink":"https://link",
  "rbac": {}
}
]`),
				Header:     map[string][]string{},
				StatusCode: http.StatusOK,
			}
			fakeRespNoEnv.Header.Add("Content-Type", "json")

			mockOcmInterface.EXPECT().GetTargetCluster(testClusterId).Return(trueClusterId, testClusterId, nil)
			mockOcmInterface.EXPECT().IsClusterHibernating(gomock.Eq(trueClusterId)).Return(false, nil).AnyTimes()
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(&testToken, nil).AnyTimes()
			mockClientUtil.EXPECT().MakeRawBackplaneAPIClient(gomock.Any()).Return(mockClient, nil)
			mockOcmInterface.EXPECT().GetTargetCluster(testClusterId).Return(trueClusterId, testClusterId, nil)
			mockClient.EXPECT().GetScriptsByCluster(gomock.Any(), trueClusterId, &bpclient.GetScriptsByClusterParams{Scriptname: &testScriptName}).Return(fakeRespNoEnv, nil)

			sut.SetArgs([]string{"describe", testScriptName, "--cluster-id", testClusterId})
			err := sut.Execute()

			Expect(err).To(BeNil())
		})

		It("should not work when backplane returns a non parsable response with 200 return", func() {
			mockOcmInterface.EXPECT().GetTargetCluster(testClusterId).Return(trueClusterId, testClusterId, nil)
			mockOcmInterface.EXPECT().IsClusterHibernating(gomock.Eq(trueClusterId)).Return(false, nil).AnyTimes()
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(&testToken, nil).AnyTimes()
			mockClientUtil.EXPECT().MakeRawBackplaneAPIClient(gomock.Any()).Return(mockClient, nil)
			fakeResp.Body = MakeIoReader("Sad")
			mockOcmInterface.EXPECT().GetTargetCluster(testClusterId).Return(trueClusterId, testClusterId, nil)
			mockClient.EXPECT().GetScriptsByCluster(gomock.Any(), trueClusterId, &bpclient.GetScriptsByClusterParams{Scriptname: &testScriptName}).Return(fakeResp, nil)

			sut.SetArgs([]string{"describe", testScriptName, "--cluster-id", testClusterId})
			err := sut.Execute()

			Expect(err).ToNot(BeNil())
		})
	})
})
