Feature: User Sign-Up

    Scenario: WaitingForApproval
        Given KIM is deployed
        When Resource is created:
        """
            apiVersion: kim.io/v1alpha1
            kind: User
            metadata:
                name: test-user
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
        """

    Scenario: Active
        Given KIM is deployed
        When  Resource is created:
        """
            apiVersion: kim.io/v1alpha1
            kind: User
            metadata:
                name: test-user
            spec:
                username: alias-name
                email: test@test.ts
                state: Active
        """
        Then Resource exists:
        """
            apiVersion: v1
            kind: ServiceAccount
            metadata:
                name: test-user
        """
