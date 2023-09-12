Feature: User Sign-Up

    Scenario: WaitingForApproval
        Given Create context namespace "test-wfa"
        And   KIM is deployed
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
        Given Create context namespace "test-active"
        And   KIM is deployed
        When Resource is created:
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
