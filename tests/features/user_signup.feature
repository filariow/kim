Feature: User Sign-Up

    Scenario: WaitingForApproval
        Given Resource is created:
        """
            apiVersion: v1
            kind: Namespace
            metadata:
                name: test
        """
        When Resource is created:
        """
            apiVersion: kim.io/v1alpha1
            kind: User
            metadata:
                name: test-user
                namespace: test
            spec:
                username: alias-name
                email: test@test.ts
                state: WaitingForApproval
        """
        Then Resource doesn't exist:
        """
            apiVersion: v1
            kind: ServiceAccount
            metadata:
                name: test-user
                namespace: test
        """
